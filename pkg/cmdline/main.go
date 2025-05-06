package cmdline

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// CommandLine represents a kernel command line builder
// that enforces uniqueness, correct syntax, and key ordering.
type CommandLine struct {
	params map[string]string
	flags  map[string]struct{}
	order  []string // track insertion order for special keys
}

// New creates a new CommandLine instance.
func New() *CommandLine {
	return &CommandLine{
		params: make(map[string]string),
		flags:  make(map[string]struct{}),
		order:  []string{},
	}
}

// AddParam adds a key=value parameter.
func (c *CommandLine) AddParam(key, value string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := validateValue(value); err != nil {
		return err
	}
	if _, exists := c.params[key]; exists {
		return fmt.Errorf("parameter '%s' already exists", key)
	}
	c.params[key] = value
	if key == "initrd" || key == "root" {
		c.order = append([]string{key}, c.order...)
	} else {
		c.order = append(c.order, key)
	}
	return nil
}

// AddFlag adds a flag (e.g., 'quiet', 'nomodeset') without value.
// If the flag contains an '=', it is treated as a key=value param.
func (c *CommandLine) AddFlag(flag string) error {
	if strings.Contains(flag, "=") {
		parts := strings.SplitN(flag, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid parameter format in flag: '%s'", flag)
		}
		return c.AddParam(parts[0], parts[1])
	}

	if err := validateKey(flag); err != nil {
		return err
	}
	if _, exists := c.flags[flag]; exists {
		return fmt.Errorf("flag '%s' already exists", flag)
	}
	c.flags[flag] = struct{}{}
	return nil
}

// String renders the full command line as a string.
func (c *CommandLine) String() string {
	var entries []string

	// First, add initrd and root in order if they exist
	if v, ok := c.params["initrd"]; ok {
		entries = append(entries, fmt.Sprintf("initrd=%s", v))
	}
	if v, ok := c.params["root"]; ok {
		entries = append(entries, fmt.Sprintf("root=%s", v))
	}

	// Then add all flags in alphabetical order
	flags := make([]string, 0, len(c.flags))
	for flag := range c.flags {
		flags = append(flags, flag)
	}
	sort.Strings(flags)
	entries = append(entries, flags...)

	// Finally add remaining parameters in alphabetical order
	params := make([]string, 0, len(c.params))
	for k, v := range c.params {
		// Skip initrd and root as they were already added
		if k == "initrd" || k == "root" {
			continue
		}
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(params)
	entries = append(entries, params...)

	return strings.Join(entries, " ")
}

func validateKey(key string) error {
	if strings.TrimSpace(key) != key || strings.ContainsAny(key, " \"'\\") {
		return fmt.Errorf("invalid key: '%s'", key)
	}
	if key == "" {
		return errors.New("key cannot be empty")
	}
	return nil
}

func validateValue(value string) error {
	if strings.TrimSpace(value) != value || strings.ContainsAny(value, " \"'\\") {
		return fmt.Errorf("invalid value: '%s'", value)
	}
	if value == "" {
		return errors.New("value cannot be empty")
	}
	return nil
}
