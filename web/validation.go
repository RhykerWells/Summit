package web

import (
	"errors"
	"reflect"
	"strings"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/bwmarrin/discordgo"
)

var (
	ErrInvalidRole          = errors.New("Invalid role")
	ErrInvalidChannel       = errors.New("Invalid channel")
	ErrInvalidValdationType = errors.New("Invalid validation type")
	ErrEmptyValue           = errors.New("Value cannot be empty")
)

func validateForm(guild *discordgo.Guild, form interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(form))
	t := v.Type()

	// Iterate through each form field
	numFormFields := v.NumField()
	for i := 0; i < numFormFields; i++ {
		// Check if we need to validate this field
		tField := t.Field(i)
		tag := tField.Tag
		validationStr := tag.Get("valid")
		if validationStr == "" {
			continue
		}

		// Retrive the validation type and any options
		validationTag := strings.Split(validationStr, ",")
		vField := v.Field(i)

		switch castValue := vField.Interface().(type) {
		case string:
			s, err := validateString(guild, validationTag, castValue)
			if err != nil {
				return err
			}

			vField.Set(reflect.ValueOf(s))
		case []string:
			s := make([]string, 0, len(castValue))
			for _, item := range castValue {
				validatedString, err := validateString(guild, validationTag, item)
				if err != nil {
					return err
				}
				s = append(s, validatedString)
			}

			vField.Set(reflect.ValueOf(s))
		case int64:
			i, err := validateInt64(castValue, validationTag)
			if err != nil {
				return err
			}

			vField.Set(reflect.ValueOf(i))
		}
	}

	return nil
}

func validateInt64(i int64, validation []string) (int64, error) {
	// Check for min/max
	var minPtr *int64
	if len(validation) > 0 {
		min := functions.ToInt64(validation[0])
		minPtr = &min
	}

	var maxPtr *int64
	if len(validation) > 1 {
		max := functions.ToInt64(validation[1])
		maxPtr = &max
	}

	if minPtr == nil {
		return i, nil
	}

	if maxPtr == nil {
		if i < *minPtr {
			return 0, errors.New("Value must be greater than or equal to " + validation[0])
		}

		return i, nil
	}

	if i < *minPtr || i > *maxPtr {
		return 0, errors.New("Value must be between " + validation[0] + " and " + validation[1])
	}

	return i, nil
}

func validateString(guild *discordgo.Guild, validation []string, value string) (string, error) {
	validationType := validation[0]

	allowEmpty := false
	if len(validation) > 1 {
		allow := validation[1]
		if allow == "allowEmpty" {
			allowEmpty = true
		}
	}

	switch validationType {
	case "role":
		return value, validateRoleField(value, guild.Roles, allowEmpty)
	case "channel":
		return value, validateChannelField(value, guild.Channels, allowEmpty)
	case "":
		if !allowEmpty && value == "" {
			return "", ErrEmptyValue
		}
		return value, nil
	}

	return "", ErrInvalidValdationType
}

func validateRoleField(role string, guildRoles []*discordgo.Role, allowempty bool) error {
	if allowempty && role == "" {
		return nil
	}

	for _, r := range guildRoles {
		if r.ID == role {
			return nil
		}
	}

	return ErrInvalidRole
}

func validateChannelField(channel string, guildChannels []*discordgo.Channel, allowempty bool) error {
	if allowempty && channel == "" {
		return nil
	}

	for _, c := range guildChannels {
		if c.ID == channel {
			return nil
		}
	}

	return ErrInvalidChannel
}
