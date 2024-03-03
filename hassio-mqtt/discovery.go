package hassiomqtt

// ConnectionModel is the tuple [connection_type,connection_identifier]
//
// An example would be ["mac", "02:5b:26:a8:dc:12"] for the MAC address of
// a network interface.
type ConnectionModel [2]string

// DeviceModel provides information about the device the entity is part of.
//
// UniqueID must be set in the entity for this information to work.
type DeviceModel struct {
	// ConfigurationURL is a link to a web page for configuring the device
	ConfigurationURL string `json:"configuration_url,omitempty"`

	// Connections of the device to the outside world, such as MAC addresses.
	Connections []ConnectionModel `json:"connections,omitempty"`

	// HardwareVersion of the device
	HardwareVersion string `json:"hw_version,omitempty"`

	// A list of IDs that uniquely identify the device (such as serial number)
	Identifiers []string `json:"identifiers,omitempty"`

	// The manufacturer of the device
	Manufacturer string `json:"manufacturer,omitempty"`

	// The model of the device
	Model string `json:"model,omitempty"`

	// The name of the device
	Name string `json:"name,omitempty"`

	// SerialNumber of the device
	SerialNumber string `json:"serial_number,omitempty"`

	// SuggestedArea if a device isn't already in one
	SuggestedArea string `json:"suggested_area,omitempty"`

	// SoftwareVersion of the device
	SoftwareVersion string `json:"sw_version,omitempty"`

	// Identifier of a device that routes messages between the device and HASS,
	// typically a 'hub' device
	ViaDevice string `json:"via_device,omitempty"`
}

type AvailabilityModel struct {
	// The value (after processing with `value_template`) indicating
	// the entity is available.
	PayloadAvailable string `json:"payload_available"`

	// The value (after processing with `availability_template`) indicating
	// the entity is not available.
	PayloadNotAvailable string `json:"payload_not_available"`

	// An MQTT topic subscribed to receive availability (online/offline) updates.
	Topic string `json:"topic"`

	// A template used to extract availability from Topic.  The result of this
	// template is compared to PayloadAvailable and PayloadNotAvailable.
	ValueTemplate string `json:"value_template"`
}

type EntityModel struct {
	// Availability controls the enable/disable state of the entity
	Availability []AvailabilityModel `json:"availability,omitempty"`

	// AvailabilityMode is one of 'all', 'any' or 'latest' indicating
	// how multiple availability specifications are combined
	AvailabilityMode string `json:"availability_mode,omitempty"`

	// Device indicates which device this entity is part of
	Device *DeviceModel `json:"device,omitempty"`

	// DeviceClass is one of the well-known HASS device classes, such as
	// 'switch', 'humidity', etc.
	DeviceClass string `json:"device_class,omitempty"`

	// EnabledByDefault determines if the entity should be enabled by default
	EnabledByDefault *bool `json:"enabled_by_default,omitempty"`

	// Encoding indicates the character encoding of MQTT payloads and messages
	Encoding string `json:"encoding,omitempty"`

	// EntityCategory is nil, "config" or "diagnostic"
	EntityCategory string `json:"entity_category,omitempty"`

	// Icon is one of the MDI icons, eg. "mdi:home"
	Icon string `json:"icon,omitempty"`

	// JSONAttributesTemplate extracts the attributes dictionary from the
	// JSONAttributesTopic.
	JSONAttributesTemplate string `json:"json_attributes_template,omitempty"`

	// JSONAttributesTopic is the topic subscribed to receive a JSON dictionary payload
	// of the entity attributes.
	JSONAttributesTopic string `json:"json_attributes_topic,omitempty"`

	// LastResetValueTemplate defines a template to extract the last time an entity with
	// StateClass `total` or `total_increasing` was reset
	LastResetValueTemplate string `json:"last_reset_value_template,omitempty"`

	// Name specifies the HASS default display name
	Name string `json:"name,omitempty"`

	// ObjectID overrides the automatic entity id in hass
	ObjectID string `json:"object_id,omitempty"`

	// QOS is the maximum QoS level to be used when receiving and publishing messages
	QOS *int `json:"qos,omitempty"`

	// StateTopic is the MQTT topic subscribed to receive values
	StateTopic string `json:"state_topic,omitempty"`

	// UniqueID is an ID uniquely identifying this entity
	UniqueID string `json:"unique_id,omitempty"`

	// ValueTemplate defines the template to extract the value
	ValueTemplate string `json:"value_template,omitempty"`
}

type SensorModel struct {
	EntityModel

	// SuggestedDisplayPrecision is the number of decimals that should be shown
	SuggestedDisplayPrecision int `json:"suggested_display_precision,omitempty"`

	// StateClass is one of 'measurement', 'total' or 'total_increasing'
	StateClass string `json:"state_class,omitempty"`

	// UnitOfMeasurement defines the measurement units of the sensor (if any)
	UnitOfMeasurement string `json:"unit_of_measurement,omitempty"`
}

type SwitchModel struct {
	EntityModel

	CommandTopic string `json:"command_topic"`

	Optimistic *bool `json:"optimistic,omitempty"`

	PayloadOff string `json:"payload_off,omitempty"`

	PayloadOn string `json:"payload_on,omitempty"`

	Retain *bool `json:"retain,omitempty"`

	StateOn string `json:"state_on,omitempty"`

	StateOff string `json:"state_off,omitempty"`
}
