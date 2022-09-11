// Package sx127x implements support for Semtech SX127x, of which the HOPERF is a common module.
//
// Datasheet: https://semtech.my.salesforce.com/sfc/p/#E0000000JelG/a/2R0000001Rbr/6EfVZUorrpoKFfvaF_Fkpgp5kzjiNyiAbqcpqh9qSjE
// Datasheet: https://www.hoperf.com/data/upload/portal/20190801/RFM95W-V2.0.pdf
//
// Adafruit has an example break-out: https://www.adafruit.com/product/3072
//
// This code is based on the Circuit Python implementation, itself adapted from
// the Radiohead library RF95 code:
//
//	https://github.com/adafruit/Adafruit_CircuitPython_RFM9x/blob/main/adafruit_rfm9x.py
package sx127x

import (
	"errors"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

// PowerMode is an 'enum' to allow high-power/low-power modes.
//
// An enum is used to detect 'default' value, which is 'high-power'.
type PowerMode uint8

const (
	PowerModeDefault = 0
	PowerModeLow     = 1
	PowerModeHigh    = 2
)

// CrcMode is an 'enum' to control Crc calculations.
//
// An enum is used to detect 'default' value, which is 'On'.
type CrcMode uint8

const (
	CrcModeDefault = 0
	CrcModeOff     = 1
	CrcModeOn      = 2
)

type irqFlags uint8

func (f irqFlags) isTxDone() bool {
	return (f & IRQ_TX_DONE) != 0
}

func (f irqFlags) isRxDone() bool {
	return (f & IRQ_RX_DONE) != 0
}

var (
	// bwBins converts between desired bandwidth (Hz) and register config (index)
	bwBins = [9]int{7800, 10400, 15600, 20800, 31250, 41700, 62500, 125000, 250000}
)

type Config struct {
	// Frequency is in Hz
	Frequency uint32

	// PowerMode enables a +20dBm mode
	PowerMode PowerMode

	// Length of LoRa preamble
	//
	// NOTE: this is the configured pre-amble length, the actual
	// pre-amble is 4 symbols greater
	PreambleLength uint16

	// CodingRate increases forward error-correction at cost of
	// reduced bit rate.  Valid values are 5,6,7,8.
	CodingRate uint8

	// SpreadingFactor increases ability to distinguish signal from
	// noise at cost of reduced bit rate.  Valid values are 6 through 12.
	SpreadingFactor uint8

	// Bandwidth is the signal bandwidth in Hz
	Bandwidth int

	// CRC controls CRC calculations
	CRC CrcMode

	// AGC controls automatic gain control (default is off)
	AGC bool
}

type Device struct {
	bus      spi.Port
	conn     spi.Conn
	resetPin gpio.PinOut
	dio0Pin  gpio.PinIn

	highPower bool
}

var (
	errNoFrequency  = errors.New("frequency mandatory")
	errNotDetected  = errors.New("sx127x not detected")
	errBadFrequency = errors.New("frequency must be between 240MHz and 960MHz")
	errTxPowerRange = errors.New("tx power outside of acceptable range")
	errPacketSize   = errors.New("packet too large")
	errTimeout      = errors.New("timeout")
	errBadCrc       = errors.New("bad CRC")
)

// New creates a new SSD1351 connection. The SPI wire must already be configured.
//
// dio0Pin is optional, if passed interrupt-style interaction will be used - or if not
// connected use `machine.NoPin` to use polling.
func New(bus spi.Port, resetPin gpio.PinOut, dio0Pin gpio.PinIn) (Device, error) {
	conn, err := bus.Connect(1*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return Device{}, err
	}

	return Device{
		bus:      bus,
		conn:     conn,
		resetPin: resetPin,
		dio0Pin:  dio0Pin,
	}, nil
}

func (d *Device) Configure(cfg Config) error {
	// It is mandatory to set frequency.  Since permitted frequencies vary by
	// region there is no good global default.
	if cfg.Frequency == 0 {
		return errNoFrequency
	}

	// Default to high power
	d.highPower = cfg.PowerMode != PowerModeLow

	frequency := cfg.Frequency

	preambleLength := cfg.PreambleLength
	if cfg.PreambleLength == 0 {
		preambleLength = 8 // default to Radiohead value
	}

	bandwidth := cfg.Bandwidth
	if cfg.Bandwidth == 0 {
		bandwidth = 125000 // default to RadioHead compatible Bw125Cr45Sf128 mode
	}

	codingRate := cfg.CodingRate
	if cfg.CodingRate == 0 {
		codingRate = 5 // default to RadioHead compatible Bw125Cr45Sf128 mode
	}

	spreadingFactor := cfg.SpreadingFactor
	if cfg.SpreadingFactor == 0 {
		spreadingFactor = 7 // default to RadioHead compatible Bw125Cr45Sf128 mode
	}

	// Default to CRC on
	crc := cfg.CRC != CrcModeOff

	gpioreg.ByName("")
	d.resetPin.Out(gpio.High)
	if d.dio0Pin != nil {
		d.dio0Pin.In(gpio.PullDown, gpio.RisingEdge)
	}

	// Reset pulses low to perform reset
	d.resetPin.Out(gpio.Low)
	d.reset()

	// Go to sleep mode to enable LoRa
	d.Sleep()
	time.Sleep(10 * time.Millisecond)
	d.setLongRangeMode(true)

	// Set low frequency mode < 525MHz, giving access to appropriate
	// "Band Specific Additional Registers" (address space from 0x61 to 0x73)
	d.setLowFrequencyMode(frequency < 525000000)

	// Use entire FIFO for rx or tx, but not at the same time
	d.writeUint8(REG_0E_FIFO_TX_BASE_ADDR, 0x00)
	d.writeUint8(REG_0F_FIFO_RX_BASE_ADDR, 0x00)

	d.Idle()

	d.setFrequency(frequency)
	d.setPreambleLength(preambleLength)
	d.setSignalBandwidth(bandwidth)
	d.setCodingRate(codingRate)
	d.setSpreadingFactor(spreadingFactor)
	d.enableCrc(crc)

	d.setRegBit(REG_26_MODEM_CONFIG3, cfg.AGC, REG_26_MODEM_CONFIG3_AUTO_AGC_OFFSET)

	// Set transmit power to 13 dBm, a safe value any module supports.
	d.setTxPower(13)

	return nil
}

func (d *Device) Detect() bool {
	// No real device detection, so check device version
	return d.readUint8(REG_42_VERSION) == 18
}

func (d *Device) Sleep() {
	d.setOperationMode(OP_MODE_SLEEP)
}

func (d *Device) Idle() {
	d.setOperationMode(OP_MODE_STANDBY)
}

// LoraTx sends a lora packet, (with timeout)
func (d *Device) LoraTx(pkt []uint8, timeoutMs uint32) error {
	if len(pkt) > 256 {
		return errPacketSize
	}

	d.setOperationMode(OP_MODE_STANDBY) // Standby required to write to FIFO
	d.writeUint8(REG_0D_FIFO_ADDR_PTR, 0)
	d.writeFrom(REG_00_FIFO, pkt)
	d.writeUint8(REG_22_PAYLOAD_LENGTH, uint8(len(pkt)))
	d.setDIO0Mapping(0b01) // Interrupt on TxDone
	d.setOperationMode(OP_MODE_TX)

	// Wait for TX to complete
	end := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	evt := d.waitForAnyEvent(1<<REG_12_IRQ_FLAGS_TXDONE_OFFSET, end)

	if evt == 0 {
		return errTimeout
	}

	return nil
}

// LoraRx tries to receive a LoRa packet (with timeout in milliseconds)
//
// On success, the result is allocated buffer, nil error
// On timeout, the result is nil buffer, nil error
// On error, the result is nil buffer, error
//
// CRC failures are considered as errors, the caller can simply re-listen
func (d *Device) LoraRx(timeoutMs uint32) ([]uint8, error) {
	return d.rx(timeoutMs, nil)
}

// LoraRxTo tries to receive a LoRa packet (with timeout in milliseconds)
//
// On success, the result is put in buf and the length returned, nil error
// On timeout, the result is zero length, nil error
// On error, the result is zero length, error
//
// CRC failures are considered as errors, the caller can simply re-listen
func (d *Device) LoraRxTo(timeoutMs uint32, buf []uint8) (int, error) {
	pkt, err := d.rx(timeoutMs, buf)
	if err != nil {
		return 0, err
	}

	return len(pkt), nil
}

// rx is the common implementation for LoraRx and LoraRxTo.
//
// buf is optional, if not specified an allocation will occur.  The
// caller should always use the returned slice since that is truncated
// to the length of the returned packet.
func (d *Device) rx(timeoutMs uint32, buf []uint8) ([]uint8, error) {
	d.setDIO0Mapping(0b00)                    // Interrupt on RxDone
	d.setOperationMode(OP_MODE_RX_CONTINUOUS) // Use RX continuous so device can listen for long periods

	end := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	evt := d.waitForAnyEvent(1<<REG_12_IRQ_FLAGS_RXDONE_OFFSET|1<<REG_12_IRQ_FLAGS_RXTIMEOUT_OFFSET, end)

	// Force idle since in continuous mode to prevent overwrite by new packets
	d.Idle()

	if !evt.isRxDone() {
		// timeout
		return nil, nil
	}

	if d.crcError() {
		return nil, errBadCrc
	}

	length := d.readUint8(REG_13_RX_NB_BYTES)
	addr := d.readUint8(REG_10_FIFO_RX_CURRENT_ADDR)
	d.writeUint8(REG_0D_FIFO_ADDR_PTR, addr)

	if buf == nil {
		buf = make([]byte, length)
	} else {
		buf = buf[:minInt(int(length), len(buf))]
	}
	d.readTo(REG_00_FIFO, buf)

	return buf, nil
}

func (d *Device) waitForAnyEvent(events uint8, end time.Time) irqFlags {
	now := time.Now()

	for now.Before(end) {
		var e uint8

		if d.dio0Pin != nil {
			edge := d.dio0Pin.WaitForEdge(end.Sub(now))
			if edge {
				e = d.readUint8(REG_12_IRQ_FLAGS)
				d.clearInterrupts()
			}
		} else {
			e = d.readUint8(REG_12_IRQ_FLAGS)
		}

		if e&events != 0 {
			// If no interrupt handler, we clear the interrupt flags
			// explicitly
			if d.dio0Pin == nil {
				d.clearInterrupts()
			}

			return irqFlags(e)
		}

		now = time.Now()
	}

	return 0
}

func (d *Device) crcError() bool {
	// CRC error if there was a CRC present on the last packet AND the CRC error
	// IRQ flag is set
	return d.getRegBit(REG_1C_HOP_CHANNEL, REG_1C_HOP_CHANNEL_CRC_ON_PAYLOAD_OFFSET) &&
		d.getRegBit(REG_12_IRQ_FLAGS, REG_12_IRQ_FLAGS_PAYLOAD_CRC_ERR_OFFSET)
}

func (d *Device) reset() {
	d.resetPin.Out(gpio.Low)
	time.Sleep(100 * time.Microsecond)
	d.resetPin.Out(gpio.High)
	time.Sleep(5 * time.Millisecond)
}

func (d *Device) clearInterrupts() {
	d.writeUint8(REG_12_IRQ_FLAGS, 0xFF)
}

func (d *Device) getOperationMode() {
	d.getRegBits(REG_01_OP_MODE, REG_01_OP_MODE_MODE_MASK, REG_01_OP_MODE_MODE_OFFSET)
}

func (d *Device) setOperationMode(mode uint8) {
	d.setRegBits(REG_01_OP_MODE, mode, REG_01_OP_MODE_MODE_MASK, REG_01_OP_MODE_MODE_OFFSET)
}

func (d *Device) getLongRangeMode() bool {
	return d.getRegBits(REG_01_OP_MODE, REG_01_OP_MODE_LRM_MASK, REG_01_OP_MODE_LRM_OFFSET) != 0
}

func (d *Device) setLongRangeMode(mode bool) {
	d.setRegBits(REG_01_OP_MODE, boolToUint8(mode), REG_01_OP_MODE_LRM_MASK, REG_01_OP_MODE_LRM_OFFSET)
}

func (d *Device) getLowFrequencyMode() bool {
	return d.getRegBits(REG_01_OP_MODE, REG_01_OP_MODE_LFM_MASK, REG_01_OP_MODE_LFM_OFFSET) != 0
}

func (d *Device) setLowFrequencyMode(mode bool) {
	d.setRegBits(REG_01_OP_MODE, boolToUint8(mode), REG_01_OP_MODE_LFM_MASK, REG_01_OP_MODE_LFM_OFFSET)
}

func (d *Device) setFrequency(freq uint32) error {
	if freq < 240000000 || freq > 960000000 {
		return errBadFrequency
	}

	// Do calculation as 64 unsigned to avoid using floating point calc
	// from the spec / Circuit Python implementation.
	frf := uint32((uint64(freq)*uint64(524288))/FXOSC) & 0xFFFFFF

	// Extract byte values and update registers.
	d.writeUint8(REG_06_FRF_MSB, uint8(frf>>16))
	d.writeUint8(REG_07_FRF_MID, uint8((frf>>8)&0xFF))
	d.writeUint8(REG_08_FRF_LSB, uint8(frf&0xFF))

	return nil
}

func (d *Device) setPreambleLength(val uint16) {
	d.writeUint8(REG_20_PREAMBLE_MSB, uint8((val>>8)&0xFF))
	d.writeUint8(REG_21_PREAMBLE_LSB, uint8(val&0xFF))
}

func (d *Device) getSignalBandwidth() int {
	return bwBins[d.getRegBits(REG_1D_MODEM_CONFIG1, REG_1D_MODEM_CONFIG1_BW_MASK, REG_1D_MODEM_CONFIG1_BW_OFFSET)]
}

func (d *Device) setSignalBandwidth(bw int) {
	bwId := 9
	for i, v := range bwBins {
		if bw <= v {
			bwId = i
			break
		}
	}

	d.setRegBits(REG_1D_MODEM_CONFIG1, uint8(bwId), REG_1D_MODEM_CONFIG1_BW_MASK, REG_1D_MODEM_CONFIG1_BW_OFFSET)

	// Implement Semtech SX127x errata work-arounds per Adafruit implementation
	if bw >= 500000 {
		d.setRegBits(REG_31_DETECTION_OPTIMIZE, 1, REG_31_DETECTION_OPTIMIZE_AUTOIFON_MASK, REG_31_DETECTION_OPTIMIZE_AUTOIFON_OFFSET)

		if d.getLowFrequencyMode() {
			d.writeUint8(REG_36_HIGHBW_OPTIMIZE1, 0x02)
			d.writeUint8(REG_3A_HIGHBW_OPTIMIZE2, 0x20)
		} else {
			d.writeUint8(REG_36_HIGHBW_OPTIMIZE1, 0x02)
			d.writeUint8(REG_3A_HIGHBW_OPTIMIZE2, 0x64)
		}
	} else {
		d.setRegBits(REG_31_DETECTION_OPTIMIZE, 0, REG_31_DETECTION_OPTIMIZE_AUTOIFON_MASK, REG_31_DETECTION_OPTIMIZE_AUTOIFON_OFFSET)

		d.writeUint8(REG_36_HIGHBW_OPTIMIZE1, 0x03)
		if bw == 7800 {
			d.writeUint8(REG_2F_IF_FREQ2, 0x48)
		} else if bw >= 62500 {
			d.writeUint8(REG_2F_IF_FREQ2, 0x40)
		} else {
			d.writeUint8(REG_2F_IF_FREQ2, 0x44)
		}
		d.writeUint8(REG_30_IF_FREQ1, 0)
	}
}

func (d *Device) setCodingRate(rate uint8) {
	denominator := min(max(rate, 5), 8)
	crId := denominator - 4
	d.setRegBits(REG_1D_MODEM_CONFIG1, crId, REG_1D_MODEM_CONFIG1_CR_MASK, REG_1D_MODEM_CONFIG1_CR_OFFSET)
}

func (d *Device) getSpreadingFactor() uint8 {
	return d.getRegBits(REG_1E_MODEM_CONFIG2, REG_1E_MODEM_CONFIG2_SF_MASK, REG_1E_MODEM_CONFIG2_SF_OFFSET)
}

func (d *Device) setSpreadingFactor(sf uint8) {

	sf = min(max(sf, 6), 12)

	var dThold uint8
	if sf == 6 {
		dThold = 0x0C
		d.setRegBits(REG_31_DETECTION_OPTIMIZE, 0x5, REG_31_DETECTION_OPTIMIZE_SF_MASK, REG_31_DETECTION_OPTIMIZE_SF_OFFSET)
	} else {
		dThold = 0x0A
		d.setRegBits(REG_31_DETECTION_OPTIMIZE, 0x3, REG_31_DETECTION_OPTIMIZE_SF_MASK, REG_31_DETECTION_OPTIMIZE_SF_OFFSET)
	}

	d.writeUint8(REG_37_DETECTION_THRESHOLD, dThold)
	d.setRegBits(REG_1E_MODEM_CONFIG2, sf, REG_1E_MODEM_CONFIG2_SF_MASK, REG_1E_MODEM_CONFIG2_SF_OFFSET)
}

func (d *Device) enableCrc(mode bool) {
	d.setRegBit(REG_1E_MODEM_CONFIG2, mode, REG_1E_MODEM_CONFIG2_CRC_OFFSET)
}

func (d *Device) isCrcEnabled() bool {
	return d.getRegBit(REG_1E_MODEM_CONFIG2, REG_1E_MODEM_CONFIG2_CRC_OFFSET)
}

func (d *Device) setTxPower(db int) error {
	if d.highPower {
		if db < 5 || db > 23 {
			return errTxPowerRange
		}

		if db > 20 {
			d.setRegBits(REG_4D_PA_DAC, PA_DAC_ENABLE, REG_4D_PA_DAC_BOOST_MASK, REG_4D_PA_DAC_BOOST_OFFSET)
			db -= 3
		} else {
			d.setRegBits(REG_4D_PA_DAC, PA_DAC_DISABLE, REG_4D_PA_DAC_BOOST_MASK, REG_4D_PA_DAC_BOOST_OFFSET)
		}

		d.setRegBit(REG_09_PA_CONFIG, true, REG_09_PA_CONFIG_SELECT_OFFSET)
		d.setRegBits(REG_09_PA_CONFIG, uint8((db-5)&0x0F), REG_09_PA_CONFIG_POWER_MASK, REG_09_PA_CONFIG_POWER_OFFSET)
	} else {
		if db < 0 || db > 14 {
			return errTxPowerRange
		}

		d.setRegBit(REG_09_PA_CONFIG, false, REG_09_PA_CONFIG_SELECT_OFFSET)
		d.setRegBits(REG_09_PA_CONFIG, 0b111, REG_09_PA_CONFIG_MAX_POWER_MASK, REG_09_PA_CONFIG_MAX_POWER_OFFSET)
		d.setRegBits(REG_09_PA_CONFIG, uint8((db+1)&0x0F), REG_09_PA_CONFIG_POWER_MASK, REG_09_PA_CONFIG_POWER_OFFSET)
	}

	return nil
}

func (d *Device) setDIO0Mapping(fn uint8) {
	d.setRegBits(REG_40_DIO_MAPPING1, fn, REG_40_DIO_MAPPING1_MASK, REG_40_DIO_MAPPING1_OFFSET)
}

func (d *Device) setRegBits(reg uint8, val uint8, mask uint8, offset uint8) {
	v := d.readUint8(reg)
	v &= ^(mask << offset)
	v |= (val & mask) << offset
	d.writeUint8(reg, v)
}

func (d *Device) getRegBits(reg uint8, mask uint8, offset uint8) uint8 {
	val := d.readUint8(reg)
	return (val >> offset) & mask
}

func (d *Device) setRegBit(reg uint8, val bool, offset uint8) {
	if val {
		d.setRegBits(reg, 1, 1, offset)
	} else {
		d.setRegBits(reg, 0, 1, offset)
	}
}

func (d *Device) getRegBit(reg uint8, offset uint8) bool {
	return d.getRegBits(reg, 1, offset) > 0
}

func (d *Device) readUint8(reg uint8) uint8 {

	// High bit clear = read
	buf := []byte{reg & 0x7F, 0}

	d.conn.Tx(buf, buf)

	return buf[1]
}

func (d *Device) writeUint8(reg uint8, data uint8) {
	// High bit set = write
	buf := []byte{reg | 0x80, data}

	d.conn.Tx(buf, buf)
}

func (d *Device) writeFrom(reg uint8, data []byte) {
	tBuf := make([]byte, len(data)+1)

	// High bit set = write
	tBuf[0] = reg | 0x80
	copy(tBuf[1:], data)

	d.conn.Tx(tBuf, nil)
}

func (d *Device) readTo(reg uint8, data []byte) {
	tBuf := make([]byte, len(data)+1)

	// Low bit clear = read
	tBuf[0] = reg & 0x7F

	d.conn.Tx(tBuf, tBuf)

	copy(data, tBuf[1:])
}

func boolToUint8(boolVal bool) uint8 {
	if boolVal {
		return 1
	}
	return 0
}

func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

func max(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
