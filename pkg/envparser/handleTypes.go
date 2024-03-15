package envparser

import (
	"fmt"
	"strconv"
)

/*
TODO: parse the secrets dynamically to it's respective types
GetEnvVarGeneric attempts to retrieve an environment variable and guess its type.
Currently it's not being used
*/
func GetEnvVarGeneric(key string) (interface{}, error) {
	valueStr, ok := envVars[key]
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %s", key)
	}

	// Attempt to parse as bool
	if valueBool, err := strconv.ParseBool(valueStr); err == nil {
		return valueBool, nil
	}

	// Attempt to parse as int
	if valueInt, err := strconv.Atoi(valueStr); err == nil {
		return valueInt, nil
	}

	// Attempt to parse as float
	if valueFloat, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return valueFloat, nil
	}

	// Default to string if no other types match
	return valueStr, nil
}
