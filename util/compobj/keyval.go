package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type (
	CompKeyvals struct {
		*Obj
	}
	CompKeyval struct {
		Key   string `json:"key"`
		Op    string `json:"op"`
		Value any    `json:"value"`
	}
)

var (
	keyValResetMap     = map[string]int{}
	keyValpath         string
	keyValFileFmtCache []byte
	keyvalValidityMap  = map[string]string{}
	compKeyvalInfo     = ObjInfo{
		DefaultPrefix: "OSVC_COMP_KEYVAL_",
		ExampleValue: []CompKeyval{{
			Key:   "PermitRootLogin",
			Op:    "=",
			Value: "yes",
		}, {
			Key:   "PermitRootLogin",
			Op:    "reset",
			Value: "",
		},
		},
		Description: `* Setup and verify keys in "key value" formatted configuration file.
* Example files: sshd_config, ssh_config, ntp.conf, ...
`,
		FormDefinition: `Desc: |
  A rule to set a list of parameters in simple keyword/value configuration file format. Current values can be checked as set or unset, or superior/inferior to their target value. By default, this object appends keyword/values not found, potentially creating duplicates. The 'reset' operator can be used to avoid such duplicates.
Outputs:
  -
    Dest: compliance variable
    Type: json
    Format: list of dict
    Class: keyval
Inputs:
  -
    Id: key
    Label: Key
    DisplayModeTrim: 64
    DisplayModeLabel: key
    LabelCss: action16
    Mandatory: Yes
    Type: string
    Help:
  -
    Id: op
    Label: Comparison operator
    DisplayModeLabel: op
    LabelCss: action16
    Mandatory: Yes
    Type: string
    Default: "="
    Candidates:
      - reset
      - unset
      - "="
      - ">="
      - "<="
      - "IN"
    Help: The comparison operator to use to check the parameter current value. The 'reset' operator can be used to avoid duplicate occurence of the same keyword (insert a key reset after the last key set to mark any additional key found in the original file to be removed). The IN operator verifies the current value is one of the target list member. On fix, if the check is in error, it sets the first target list member. A "IN" operator value must be a JSON formatted list.
  -
    Id: value
    Label: Value
    DisplayModeLabel: value
    LabelCss: action16
    Mandatory: Yes
    Type: string or integer
    Help: The configuration file parameter target value.
`,
	}
)

func init() {
	m["keyval"] = NewCompKeyvals
}

func NewCompKeyvals() interface{} {
	return &CompKeyvals{
		Obj: NewObj(),
	}
}

func (t *CompKeyvals) Add(s string) error {
	dataPath := struct {
		Path string       `json:"path"`
		Keys []CompKeyval `json:"keys"`
	}{}
	if err := json.Unmarshal([]byte(s), &dataPath); err != nil {
		return err
	}
	if dataPath.Path == "" {
		t.Errorf("path should be in the dict: %s\n", s)
		return fmt.Errorf("path should be in the dict: %s\n", s)
	}
	keyValpath = dataPath.Path
	for _, rule := range dataPath.Keys {
		if rule.Key == "" {
			t.Errorf("key should be in the dict: %s\n", s)
			return fmt.Errorf("key should be in the dict: %s\n", s)
		}
		if rule.Op == "" {
			rule.Op = "="
		}
		if rule.Value == nil && (rule.Op == "=" || rule.Op == ">=" || rule.Op == "<=" || rule.Op == "IN") {
			t.Errorf("value should be set in the dict: %s\n", s)
			return fmt.Errorf("value should be set in the dict: %s\n", s)
		}
		if !(rule.Op == "reset" || rule.Op == "unset" || rule.Op == "=" || rule.Op == ">=" || rule.Op == "<=" || rule.Op == "IN") {
			t.Errorf("op should be in: reset, unset, =, >=, <=, IN in dict: %s\n", s)
			return fmt.Errorf("op should be in: reset, unset, =, >=, <=, IN in dict: %s\n", s)
		}
		if rule.Op != "unset" {
			switch rule.Value.(type) {
			case string:
			//skip
			case float64:
			//skip
			default:
				if rule.Op != "IN" {
					t.Errorf("value should be an int or a string in dict: %s\n", s)
					return fmt.Errorf("value should be an int or a string in dict: %s\n", s)
				}
				if _, ok := rule.Value.([]any); !ok {
					t.Errorf("value should be a list in dict: %s\n", s)
					return fmt.Errorf("value should be a list in dict: %s\n", s)
				}
				for _, val := range rule.Value.([]any) {
					if _, ok := val.(float64); ok {
						continue
					}
					if _, ok := val.(string); ok {
						continue
					}
					t.Errorf("the values in value list should be string or int in dict: %s\n", s)
					return fmt.Errorf("the values in value should be string or int in dict: %s\n", s)
				}
			}
		}

		if _, ok := rule.Value.(float64); !ok && (rule.Op == "<=" || rule.Op == ">=") {
			t.Errorf("can't use >= and <= if the value is not an int in dict: %s\n", s)
			return fmt.Errorf("can't use >= and <= if the value is not an int in dict: %s\n", s)
		}

		switch rule.Op {
		case "unset":
			if keyvalValidityMap[rule.Key] == "set" {
				keyvalValidityMap[rule.Key] = "unValid"
			} else if keyvalValidityMap[rule.Key] != "unValid" {
				keyvalValidityMap[rule.Key] = "unset"
			}
		case "reset":
			keyValResetMap[rule.Key] = 0
		default:
			if keyvalValidityMap[rule.Key] == "unset" {
				keyvalValidityMap[rule.Key] = "unValid"
			} else if keyvalValidityMap[rule.Key] != "unValid" {
				keyvalValidityMap[rule.Key] = "set"
			}
		}
		t.Obj.Add(rule)
	}
	t.filterRules()
	return nil
}

func (t *CompKeyvals) filterRules() {
	blacklisted := map[string]any{}
	newRules := []interface{}{}
	for _, rule := range t.Rules() {
		if keyvalValidityMap[rule.(CompKeyval).Key] != "unValid" {
			newRules = append(newRules, rule)
		} else if _, ok := blacklisted[rule.(CompKeyval).Key]; !ok {
			blacklisted[rule.(CompKeyval).Key] = nil
			t.Errorf("the key %s generate some conflicts (asking for a comparison operator and unset at the same time) the key is now blacklisted\n", rule.(CompKeyval).Key)
		}
	}
	t.Obj.rules = newRules
}

func (t CompKeyvals) loadCache() error {
	var err error
	keyValFileFmtCache, err = os.ReadFile(keyValpath)
	return err
}

func (t CompKeyvals) getValues(key string) []string {
	values := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(keyValFileFmtCache))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		splitLine := strings.SplitN(line, "=", 1)
		if len(splitLine) != 2 {
			continue
		}
		if splitLine[0] == key {
			values = append(values, splitLine[1])
		}
	}
	return values
}

func (t CompKeyvals) checkOperator(rule CompKeyval, valuesFromFile []string) int {
	switch rule.Op {
	case "=":
		for i, value := range valuesFromFile {
			if _, ok := rule.Value.(float64); ok {
				floatValue, err := strconv.ParseFloat(value, 64)
				if err != nil {
					continue
				}
				if rule.Value.(float64) == floatValue {
					return i
				}
			} else {
				if rule.Value.(string) == value {
					return i
				}
			}
		}
	case ">=":
		for i, value := range valuesFromFile {
			floatValueFromFile, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			if float64(rule.Value.(int)) >= floatValueFromFile {
				return i
			}
		}
	case "<=":
		for i, value := range valuesFromFile {
			floatValueFromFile, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			if float64(rule.Value.(int)) <= floatValueFromFile {
				return i
			}
		}
	case "IN":
		for _, val := range rule.Value.([]any) {
			for i, valueFromFile := range valuesFromFile {
				if _, ok := val.(float64); ok {
					floatValue, err := strconv.ParseFloat(valueFromFile, 64)
					if err != nil {
						continue
					}
					if val.(float64) == floatValue {
						return i
					}
				} else {
					if val.(string) == valueFromFile {
						return i
					}
				}
			}
		}
	}
	return -1
}

func (t CompKeyvals) checkRule(rule CompKeyval) ExitCode {
	switch rule.Op {
	case "reset":
		valuesFromFile := t.getValues(rule.Key)
		if len(valuesFromFile) != keyValResetMap[rule.Key] {
			t.VerboseErrorf("%s: %s is set %d times, should be set %d times", keyValpath, rule.Key, len(valuesFromFile), keyValResetMap[rule.Key])
			return ExitNok
		}
		t.VerboseErrorf("%s: %s is set %d times, on target", keyValpath, rule.Key, keyValResetMap[rule.Key])
		return ExitOk
	default:
		if _, ok := keyValResetMap[rule.Key]; ok {
			keyValResetMap[rule.Key] += 1
		}
		return t.checkNoReset(rule)
	}
}

func (t CompKeyvals) checkNoReset(rule CompKeyval) ExitCode {
	valuesFromFile := t.getValues(rule.Key)
	if rule.Op == "unset" {
		if len(valuesFromFile) > 0 {
			t.VerboseErrorf("%s: %s is set and should not be set", keyValpath, rule.Key)
			return ExitNok
		}
		t.VerboseErrorf("%s: %s is not set and should not be set", keyValpath, rule.Key)
		return ExitOk
	}
	if len(valuesFromFile) < 1 {
		t.VerboseErrorf("%s: %s is unset and should be set", keyValpath, rule.Key)
		return ExitNok
	}
	switch rule.Op {
	case "=":
		i := t.checkOperator(rule, valuesFromFile)
		if i == -1 {
			t.VerboseErrorf("%s: %s has the following values: %s and one of these values should be equal to %s", keyValpath, rule.Key, valuesFromFile, rule.Value)
			return ExitNok
		}
		t.VerboseErrorf("%s: %s has the following values: %s and one of these values should be equal to %s", keyValpath, rule.Key, valuesFromFile, rule.Value)
		return ExitOk
	case ">=":
		i := t.checkOperator(rule, valuesFromFile)
		if i == -1 {
			t.VerboseErrorf("%s: %s has the following values: %s and one of these values should be greater or equal to %s", keyValpath, rule.Key, valuesFromFile, rule.Value)
			return ExitNok
		}
		t.VerboseErrorf("%s: %s has the following values: %s and one of these values should be greater or equal to %s", keyValpath, rule.Key, valuesFromFile, rule.Value)
		return ExitOk
	case "<=":
		i := t.checkOperator(rule, valuesFromFile)
		if i == -1 {
			t.VerboseErrorf("%s: %s has the following values: %s and one of these values should be less or equal to %s", keyValpath, rule.Key, valuesFromFile, rule.Value)
			return ExitNok
		}
		t.VerboseErrorf("%s: %s has the following values: %s and one of these values should be less or equal to %s", keyValpath, rule.Key, valuesFromFile, rule.Value)
		return ExitOk
	default:
		i := t.checkOperator(rule, valuesFromFile)
		if i == -1 {
			t.VerboseErrorf("%s: %s has the following values: %s and one of these values should be in %s", keyValpath, rule.Key, valuesFromFile, rule.Value)
			return ExitNok
		}
		t.VerboseErrorf("%s: %s has the following values: %s and one of these values should be in %s", keyValpath, rule.Key, valuesFromFile, rule.Value)
		return ExitOk
	}
}

func (t CompKeyvals) Check() ExitCode {
	t.SetVerbose(true)
	if err := t.loadCache(); err != nil {
		t.Errorf("%s\n", err)
		return ExitNok
	}
	e := ExitOk
	/*	for _, i := range t.Rules() {
		rule := i.(CompSymlink)
		o := t.CheckSymlink(rule)
		e = e.Merge(o)
	}*/
	return e
}

func (t CompKeyvals) Fix() ExitCode {
	t.SetVerbose(false)
	e := ExitOk
	/*	for _, i := range t.Rules() {
		rule := i.(CompSymlink)
		e = e.Merge(t.fixSymlink(rule))
	}*/
	return e
}

func (t CompKeyvals) Fixable() ExitCode {
	return ExitNotApplicable
}

func (t CompKeyvals) Info() ObjInfo {
	return compKeyvalInfo
}
