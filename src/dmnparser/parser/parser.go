package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"flag"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Struct to match XML structure
type Definitions struct {
	XMLName   xml.Name   `xml:"definitions"`
	Decisions []Decision `xml:"decision"`
}

type Decision struct {
	ID            string        `xml:"id,attr"`
	Name          string        `xml:"name,attr"`
	DecisionTable DecisionTable `xml:"decisionTable"`
}

type DecisionTable struct {
	Inputs  []Input  `xml:"input"`
	Outputs []Output `xml:"output"`
	Rules   []Rule   `xml:"rule"`
}

type Input struct {
	ID              string          `xml:"id,attr"`
	Label           string          `xml:"label,attr"`
	InputExpression InputExpression `xml:"inputExpression"`
}

type InputExpression struct {
	ID      string `xml:"id,attr"`
	TypeRef string `xml:"typeRef,attr"`
	Text    string `xml:"text"`
}

type Output struct {
	ID      string `xml:"id,attr"`
	Label   string `xml:"label,attr"`
	Name    string `xml:"name,attr"`
	TypeRef string `xml:"typeRef,attr"`
}

type Rule struct {
	ID           string        `xml:"id,attr"`
	InputEntries []InputEntry  `xml:"inputEntry"`
	OutputEnties []OutputEntry `xml:"outputEntry"`
}

type InputEntry struct {
	ID   string `xml:"id,attr"`
	Text string `xml:"text"`
}

type OutputEntry struct {
	ID   string `xml:"id,attr"`
	Text string `xml:"text"`
}

func main() {
	xmlPath := flag.String("xml", "", "path to DMN XML file")
	flag.Parse()

	if *xmlPath == "" {
		fmt.Println("Error: -xml is required")
		os.Exit(1)
	}

	xmlFile, err := os.Open(*xmlPath)
	if err != nil {
		fmt.Println("Error opening XML:", err)
		os.Exit(1)
	}
	defer xmlFile.Close()

	byteValue, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		fmt.Println("Error reading XML:", err)
		os.Exit(1)
	}

	var definitions Definitions
	if err := xml.Unmarshal(byteValue, &definitions); err != nil {
		fmt.Println("Error unmarshalling XML:", err)
		os.Exit(1)
	}

	generateDecisionFunction(definitions)
}
// Helper function to generate Go code dynamically + registry
func generateDecisionFunction(defs Definitions) {
	// methodName -> receiverType for THIS RUN
	newEntries := make(map[string]string)

	// Iterate over decisions and generate Go functions
	for _, decision := range defs.Decisions {
		var functionCode string
		var imports []string
		var returnTypes []string

		// Function/method naming from DMN decision name
		functionName := toCamelCase(decision.Name)

		// Unique receiver type per decision
		receiverType := functionName + "Decision"
		receiverVar := "d"

		// Track for registry
		newEntries[functionName] = receiverType

		// Collect input labels to check for necessary imports (kept as-is)
		var inputLabels []string
		for _, input := range decision.DecisionTable.Inputs {
			inputLabels = append(inputLabels, input.Label)
		}

		// Check for required imports
		if contains(inputLabels, "DateTime") {
			fmt.Println("DateTime input found")
			imports = append(imports, "\"time\"")
		}

		// Determine output types & import math only once if needed
		needsMathImport := false
		for _, output := range decision.DecisionTable.Outputs {
			outputType := "string" // Default type is string

			switch output.TypeRef {
			case "number":
				outputType = "float64"
				needsMathImport = true
			case "boolean":
				outputType = "bool"
			}

			returnTypes = append(returnTypes, outputType)
		}

		if needsMathImport {
			imports = append(imports, "\"math\"")
		}

		// Build import block (avoid empty block)
		importBlock := ""
		if len(imports) > 0 {
			importBlock = fmt.Sprintf("import (\n    %s\n)\n", strings.Join(imports, "\n    "))
		}

		// IMPORTANT: package name must match your Go module import:
		// decisionFunctions "sparta/src/decisionfunctions"
		functionCode = fmt.Sprintf(
			"package decisionfunctions\n\n%s\ntype %s struct{}\n\nfunc (%s %s) %s(inputs map[string]interface{}) (%s) {\n",
			importBlock,
			receiverType,
			receiverVar, receiverType,
			functionName,
			strings.Join(returnTypes, ", "),
		)

		// Iterate over rules and generate conditions based on input entries
		for _, rule := range decision.DecisionTable.Rules {
			conditionsMap := make(map[string]string)
			for i, inputEntry := range rule.InputEntries {
				inputLabel := decision.DecisionTable.Inputs[i].Label 
				inputType := decision.DecisionTable.Inputs[i].InputExpression.TypeRef
				conditionsMap[inputLabel] = inputEntry.Text + "|" + inputType
			}

			condition := generateConditions(conditionsMap)

			// Generate return values dynamically
			returnValues := []string{}
			for j, outputEntry := range rule.OutputEnties {
				if decision.DecisionTable.Outputs[j].TypeRef == "number" {
					returnValues = append(returnValues, outputEntry.Text)
				} else {
					// Keep DMN literal as-is (strings already come with quotes in your XML)
					returnValues = append(returnValues, fmt.Sprintf(`%s`, outputEntry.Text))
				}
			}

			functionCode += fmt.Sprintf("    if %s {\n        return %s\n    }\n",
				condition, strings.Join(returnValues, ", "))
		}

		// Default return statement for all outputs
		defaultReturn := make([]string, len(returnTypes))
		for i, outputType := range returnTypes {
			if outputType == "float64" {
				defaultReturn[i] = "math.NaN()"
			} else if outputType == "bool" {
				defaultReturn[i] = "false"
			} else {
				defaultReturn[i] = `""`
			}
		}
		functionCode += fmt.Sprintf("    return %s\n}\n", strings.Join(defaultReturn, ", "))

		// Write the generated decision function
		writeGeneratedFunction(functionCode, functionName+".go")
	}

	// Merge + write registry
	if err := writeDecisionRegistryAccumulating(newEntries); err != nil {
		fmt.Println("Error writing DecisionRegistry:", err)
		return
	}
	fmt.Println("Generated/updated DecisionRegistry at ../../decisionfunctions/registry.go")
}

// Reads existing registry.go (if present), merges with newEntries, writes back.
func writeDecisionRegistryAccumulating(newEntries map[string]string) error {
	dir := "../../decisionfunctions"
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	regPath := filepath.Join(dir, "registry.go")

	// 1) Load existing entries (if file exists)
	existing := make(map[string]string)
	if b, err := os.ReadFile(regPath); err == nil {
		parsed, perr := parseExistingRegistry(string(b))
		if perr != nil {
			// If parsing fails, don’t destroy the old file silently
			return fmt.Errorf("failed to parse existing registry.go: %w", perr)
		}
		for k, v := range parsed {
			existing[k] = v
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error reading existing registry.go: %w", err)
	}

	// 2) Merge (new overwrites existing if same key)
	for k, v := range newEntries {
		existing[k] = v
	}

	// 3) Write merged registry (sorted)
	keys := make([]string, 0, len(existing))
	for k := range existing {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("package decisionfunctions\n\n")
	sb.WriteString("// Auto-generated. DO NOT EDIT.\n")
	sb.WriteString("// Maps decision method name -> receiver instance.\n")
	sb.WriteString("var DecisionRegistry = map[string]interface{}{\n")
	for _, methodName := range keys {
		recvType := existing[methodName]
		sb.WriteString(fmt.Sprintf("\t%q: %s{},\n", methodName, recvType))
	}
	sb.WriteString("}\n")

	return os.WriteFile(regPath, []byte(sb.String()), 0644)
}

// Parses lines like: "PatientPriority": PatientPriorityDecision{},
func parseExistingRegistry(contents string) (map[string]string, error) {
	out := make(map[string]string)

	// tolerate spaces/tabs; method is quoted; receiver is an identifier; ends with {} (optional spaces)
	re := regexp.MustCompile(`(?m)^\s*"([^"]+)"\s*:\s*([A-Za-z_][A-Za-z0-9_]*)\s*\{\}\s*,?\s*$`)
	matches := re.FindAllStringSubmatch(contents, -1)

	// If registry.go exists but we match nothing, it might be malformed or edited.
	// We treat this as an error to avoid wiping content accidentally.
	if strings.Contains(contents, "DecisionRegistry") && len(matches) == 0 {
		return nil, fmt.Errorf("no registry entries matched (file may be malformed or manually edited)")
	}

	for _, m := range matches {
		method := m[1]
		recv := m[2]
		out[method] = recv
	}
	return out, nil
}

func toCamelCase(s string) string {
	words := strings.Fields(s)
	for i := range words {
		words[i] = strings.Title(words[i])
	}
	return strings.Join(words, "")
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// Example function that iterates over the map and generates conditions
func generateConditions(inputMap map[string]string) string {
	conditions := []string{}

	for key, valueType := range inputMap {
		parts := strings.Split(valueType, "|")
		value := parts[0]
		typeRef := parts[1]

		var condition string
		if value == "" {
			condition = ""
		} else {
			switch typeRef {
			case "dateTime":
				condition = processTimeCondition(key, value)
			case "boolean":
				condition = fmt.Sprintf("inputs[\"%s\"].(bool) == %s", key, value)
			case "string":
				condition = processStringCondition(key, value)
			case "number":
				condition = processNumberCondition(key, value)
			case "context":
				condition = processContext(key, value)
			default:
				condition = fmt.Sprintf("inputs[\"%s\"].(string) == %s", key, value)
			}
		}
		if condition != "" {
			conditions = append(conditions, condition)
		}
	}
	return strings.Join(conditions, " && ")
}

func processStringCondition(key string, value string) string {
	if strings.Contains(value, ",") {
		seasons := strings.Split(value, ",")
		var orConditions []string

		for _, season := range seasons {
			trimmedSeason := strings.Trim(season, " \"")
			orConditions = append(orConditions, fmt.Sprintf("inputs[\"%s\"].(string) == \"%s\"", key, trimmedSeason))
		}
		return fmt.Sprintf("(%s)", strings.Join(orConditions, " || "))
	}
	return fmt.Sprintf("inputs[\"%s\"].(string) == \"%s\"", key, strings.Trim(value, "\""))
}

func processContext(key, value string) string {
	// If the test is an interval like [1..10], reuse interval logic
	if strings.Contains(value, "..") {
		return processIntervalCondition(key, value)
	}

	operator := extractOperator(value)
	numValue := strings.TrimSpace(strings.Trim(value, "<=>"))

	if operator == "" {
		return fmt.Sprintf("inputs[%q].(float64) == %s", key, numValue)
	}
	return fmt.Sprintf("inputs[%q].(float64) %s %s", key, operator, numValue)
}

// Helper function to parse "date and time" strings into time.Time
func extractTime(dateTimeStr string) string {
	trimmed := strings.TrimPrefix(dateTimeStr, "date and time(\"")
	trimmed = strings.TrimSuffix(trimmed, "\")")
	parsedTime, _ := time.Parse(time.RFC3339, trimmed)

	return fmt.Sprintf("time.Date(%d, time.Month(%d), %d, %d, %d, %d, 0, time.UTC)",
		parsedTime.Year(), parsedTime.Month(), parsedTime.Day(),
		parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second())
}

func processTimeCondition(key, value string) string {
	var result string
	var operator string

	if strings.Contains(value, "..") {
		parts := strings.Split(value, "..")
		leftPart := strings.TrimSpace(parts[0])
		leftPart = strings.TrimPrefix(leftPart, "[")
		rightPart := strings.TrimSpace(parts[1])
		rightPart = strings.TrimSuffix(rightPart, "]")

		leftTime := extractTime(leftPart)
		rightTime := extractTime(rightPart)

		result = fmt.Sprintf(`(inputs["%s"].(time.Time).After(%s) || inputs["%s"].(time.Time).Equal(%s)) && (inputs["%s"].(time.Time).Before(%s) || inputs["%s"].(time.Time).Equal(%s))`,
			key, leftTime, key, leftTime, key, rightTime, key, rightTime)
	} else {
		if strings.HasPrefix(value, "<=") {
			operator = "<="
			value = strings.TrimSpace(strings.TrimPrefix(value, "<="))
		} else if strings.HasPrefix(value, ">=") {
			operator = ">="
			value = strings.TrimSpace(strings.TrimPrefix(value, ">="))
		} else if strings.HasPrefix(value, "<") {
			operator = "<"
			value = strings.TrimSpace(strings.TrimPrefix(value, "<"))
		} else if strings.HasPrefix(value, ">") {
			operator = ">"
			value = strings.TrimSpace(strings.TrimPrefix(value, ">"))
		} else {
			operator = "=="
		}

		formattedTime := extractTime(value)

		switch operator {
		case "<":
			result = fmt.Sprintf(`inputs["%s"].(time.Time).Before(%s)`, key, formattedTime)
		case "<=":
			result = fmt.Sprintf(`inputs["%s"].(time.Time).Before(%s) || inputs["%s"].(time.Time).Equal(%s)`, key, formattedTime, key, formattedTime)
		case ">":
			result = fmt.Sprintf(`inputs["%s"].(time.Time).After(%s)`, key, formattedTime)
		case ">=":
			result = fmt.Sprintf(`inputs["%s"].(time.Time).After(%s) || inputs["%s"].(time.Time).Equal(%s)`, key, formattedTime, key, formattedTime)
		default:
			result = fmt.Sprintf(`inputs["%s"].(time.Time).Equal(%s)`, key, formattedTime)
		}
	}

	return result
}

func extractOperator(value string) string {
	if strings.HasPrefix(value, "<=") {
		return "<="
	} else if strings.HasPrefix(value, ">=") {
		return ">="
	} else if strings.HasPrefix(value, "<") {
		return "<"
	} else if strings.HasPrefix(value, ">") {
		return ">"
	} else if strings.HasPrefix(value, "=") {
		return "="
	}
	return ""
}

func processNumberCondition(key, value string) string {
	if strings.Contains(value, "..") {
		return processIntervalCondition(key, value)
	}

	operator := extractOperator(value)
	numValue := strings.Trim(value, "<=>")

	if operator == "" {
		return fmt.Sprintf("inputs[\"%s\"].(float64) == %s", key, numValue)
	}
	return fmt.Sprintf("inputs[\"%s\"].(float64) %s %s", key, operator, numValue)
}

func processIntervalCondition(key, value string) string {
	var lowerInclusive, upperInclusive bool

	if strings.HasPrefix(value, "[") {
		lowerInclusive = true
	} else if strings.HasPrefix(value, "]") {
		lowerInclusive = false
	}
	if strings.HasSuffix(value, "]") {
		upperInclusive = true
	} else if strings.HasSuffix(value, "[") {
		upperInclusive = false
	}

	interval := value[1 : len(value)-1]
	bounds := strings.Split(interval, "..")
	if len(bounds) != 2 {
		return ""
	}

	lowerBound := strings.TrimSpace(bounds[0])
	upperBound := strings.TrimSpace(bounds[1])

	if strings.Contains(key, ":") {
		trimmed := strings.Trim(key, "{}")
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return fmt.Sprintf("// Error: Invalid context expression: %s", key)
		}
		key = strings.TrimSpace(parts[0])
	}

	var condition string
	if lowerInclusive {
		condition += fmt.Sprintf("inputs[\"%s\"].(float64) >= %s", key, lowerBound)
	} else {
		condition += fmt.Sprintf("inputs[\"%s\"].(float64) > %s", key, lowerBound)
	}

	condition += " && "

	if upperInclusive {
		condition += fmt.Sprintf("inputs[\"%s\"].(float64) <= %s", key, upperBound)
	} else {
		condition += fmt.Sprintf("inputs[\"%s\"].(float64) < %s", key, upperBound)
	}

	return condition
}

// Helper function to write the generated Go function to a file
func writeGeneratedFunction(functionCode, filename string) {
	dir := "../../decisionfunctions"
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}

	filePath := filepath.Join(dir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(functionCode); err != nil {
		fmt.Println("Error writing to file:", err)
	}
	fmt.Printf("Generated function written to %s\n", filePath)
}