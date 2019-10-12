package parsepy

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type ParamType string

const (
	ParamTypeInt   = "int"
	ParamTypeFloat = "float"
	ParamTypeStr   = "str"
)

type Param struct {
	Name         string    `json:"name"`
	Doc          string    `json:"doc"`
	Type         ParamType `json:"type"`
	DefaultValue string    `json:"default_value"`
}

type Function struct {
	Def    string   `json:"def"`
	Doc    string   `json:"doc"`
	Params []*Param `json:"params"`
}

type Class struct {
	Def  string `json:"def"`
	Name string `json:"name"`
	Doc  string `json:"doc"`

	InitFunc Function `json:"init_func"`
}

// GetClasses return class info in a python script
// python script should be pep8
func GetClasses(filename string) ([]*Class, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("parsepy.GetClasses: %s", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := []string{}

	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), " 	"))
	}

	type blockID int
	const (
		nilBlock blockID = iota
		otherBlock
		classBlock
		initFuncBlock
	)
	ret := []*Class{}
	rootLevelBlock := nilBlock // level=1 belongs to?
	nextLevelBlock := nilBlock // level=2 belongs to?

	putDocString := func(level int, docString string) {
		if level == 1 &&
			rootLevelBlock == classBlock &&
			nextLevelBlock == nilBlock {
			backPtr := len(ret) - 1
			ret[backPtr].Doc = docString
		} else if level == 2 &&
			rootLevelBlock == classBlock &&
			nextLevelBlock == initFuncBlock {
			backPtr := len(ret) - 1
			ret[backPtr].InitFunc.Doc = docString
		}
	}

	for ptr := 0; ptr < len(lines); ptr++ {
		level, line := levelString(lines[ptr])
		if strings.TrimSpace(line) == "" {
			continue
		}

		if level == 0 {
			if isBlockStart(line) {
				if strings.HasPrefix(line, "class ") {
					class := Class{
						Name: line[6 : len(line)-1],
						Def:  line,
					}
					ret = append(ret, &class)
					rootLevelBlock = classBlock
					nextLevelBlock = nilBlock
					continue
				} else {
					rootLevelBlock = otherBlock
					continue
				}
			}
		} else if level == 1 {
			if strings.HasPrefix(line, "def __init__") {
				if rootLevelBlock == classBlock {
					backPtr := len(ret) - 1
					ret[backPtr].InitFunc.Def = line
					ret[backPtr].InitFunc.Params, _ = paramsFromInitDef(line)
					nextLevelBlock = initFuncBlock
				}
				continue
			} else if isBlockStart(line) {
				nextLevelBlock = otherBlock
				continue
			}
		}
		if strings.HasPrefix(line, `"""`) {
			// Comment
			if strings.Count(line, `"`) >= 6 && strings.HasSuffix(line, `"""`) {
				// One-line comment
				docString := line[3 : len(line)-3]
				putDocString(level, docString)
				continue
			}
			// Mulitple-lines comment
			docString := line[3:] + "\n"
			for {
				ptr++
				_, line := levelString(lines[ptr])
				if strings.HasPrefix(line, `"""`) {
					break
				}
				docString += line + "\n"
			}
			putDocString(level, docString)
		}
	}
	for _, class := range ret {
		paramDocFrmoFunctionDoc(&class.InitFunc)
	}
	return ret, nil
}

func levelString(line string) (int, string) {
	level := 0
	for strings.HasPrefix(line, "    ") {
		level++
		line = line[4:]
	}
	return level, line
}

func isBlockStart(line string) bool {
	startID := []string{"if", "def", "class", "for", "while"}
	for _, id := range startID {
		if strings.HasPrefix(line, id+" ") {
			return true
		}
	}
	return false
}

func paramsFromInitDef(def string) ([]*Param, error) {
	// def __init__(self, ...):
	argStrs := strings.Split(def[13:len(def)-2], ",")
	params := []*Param{}
	for _, argStr := range argStrs[1:] {
		var param Param
		hasType := strings.Count(argStr, ":") > 0
		hasDefaultValue := strings.Count(argStr, "=") > 0
		if hasType && hasDefaultValue {
			part1 := strings.Split(argStr, ":")
			part2 := strings.Split(part1[1], "=")
			param = Param{
				Name:         strings.TrimSpace(part1[0]),
				Type:         getType(strings.TrimSpace(part2[0])),
				DefaultValue: strings.TrimSpace(part2[1]),
			}
		} else if hasType {
			part1 := strings.Split(argStr, ":")
			tp := getType(strings.TrimSpace(part1[1]))
			param = Param{
				Name:         strings.TrimSpace(part1[0]),
				Type:         tp,
				DefaultValue: typeDefault(tp),
			}
		} else if hasDefaultValue {
			part1 := strings.Split(argStr, "=")
			defaultValue := strings.TrimSpace(part1[1])
			param = Param{
				Name:         strings.TrimSpace(part1[0]),
				Type:         getTypeByValue(defaultValue),
				DefaultValue: defaultValue,
			}
		} else {
			param = Param{
				Name:         strings.TrimSpace(argStr),
				Type:         ParamTypeInt,
				DefaultValue: "0",
			}
		}
		params = append(params, &param)
	}
	return params, nil
}

func paramDocFrmoFunctionDoc(f *Function) {
	lines := strings.Split(f.Doc, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		// Should be :param *: *
		if len(parts) != 3 {
			continue
		}
		if !(parts[0] == "" && strings.HasPrefix(parts[1], "param ")) {
			continue
		}
		name := parts[1][6:]

		for _, param := range f.Params {
			if param.Name == name {
				param.Doc = strings.TrimSpace(parts[2])
				break
			}
		}
	}
}

func getType(t string) ParamType {
	switch t {
	case "str":
		return ParamTypeStr
	case "int":
		return ParamTypeInt
	case "float":
		return ParamTypeFloat
	default:
		return ParamTypeStr
	}
}

func getTypeByValue(value string) ParamType {
	if strings.HasPrefix(value, `"`) ||
		strings.HasPrefix(value, `'`) {
		return ParamTypeStr
	}
	if strings.Count(value, ".") > 0 {
		return ParamTypeFloat
	}
	return ParamTypeInt
}

func typeDefault(t ParamType) string {
	switch t {
	case ParamTypeFloat:
		return "0.0"
	case ParamTypeInt:
		return "0"
	case ParamTypeStr:
		return ""
	}
	return ""
}
