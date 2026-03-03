package isgonHelper

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	blockchain "sparta/src/utils/interaction"
	"sparta/src/utils/ipfs"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/expr-lang/expr"
	shell "github.com/ipfs/go-ipfs-api"
)

func LoadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	// Read the private key file
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading private key file: %w", err)
	}

	// Decode the PEM block
	block, _ := pem.Decode(keyBytes)
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key file: no PEM block found or incorrect type")
	}

	// Parse the private key
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}

	return privateKey, nil
}

func GenerateDeterministicDHKeyPair(seed []byte) (*ecdh.PrivateKey, *ecdh.PublicKey, error) {
	// Use the X25519 curve for Diffie-Hellman
	curve := ecdh.X25519()

	// Hash the seed to derive a deterministic private key
	hashedSeed := sha256.Sum256(seed)

	// Create the private key deterministically
	privateKey, err := curve.NewPrivateKey(hashedSeed[:])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create private key: %w", err)
	}

	// Compute the public key from the private key
	publicKey := privateKey.PublicKey()

	return privateKey, publicKey, nil
}

func LoadPEM(filename string) ([]byte, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block from file: %s", filename)
	}
	return block.Bytes, nil
}

func ParseAttributes(input string) map[string]interface{} {
	// Split the input string by commas to separate each key-value pair
	pairs := strings.Split(input, ",")
	attributes := make(map[string]interface{})

	for _, pair := range pairs {
		// Split each key-value pair by "="
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			fmt.Printf("Invalid key-value pair: %s\n", pair)
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		// Try to parse the value into its appropriate type
		if intValue, err := strconv.Atoi(value); err == nil {
			attributes[key] = intValue // Store as integer
		} else if boolValue, err := strconv.ParseBool(value); err == nil {
			attributes[key] = boolValue // Store as boolean
		} else {
			attributes[key] = value // Store as string
		}
	}

	return attributes
}

func GenerateRandomness() (int64, error) {
	var randomness int64
	err := binary.Read(rand.Reader, binary.LittleEndian, &randomness)
	if err != nil {
		return 0, fmt.Errorf("error generating randomness: %v", err)
	}
	return randomness, nil
}

type IPFSData struct {
	Ciphertext interface{} `json:"ciphertext"`
	Extension  interface{} `json:"extension"`
}

func RetrieveCiphertext(messageID int64, functionName string) IPFSData {
	sh := shell.NewShell("localhost:5001")

	retrieverHash, _ := blockchain.GetDocument(functionName, messageID)

	data, err := ipfs.FetchDataFromIPFS(sh, retrieverHash)
	if err != nil {
		fmt.Println("Error fetching data from IPFS:", err)
		return IPFSData{}
	}

	var ipfsData IPFSData

	//var ipfsData map[string]interface{}
	if err := json.Unmarshal(data, &ipfsData); err != nil {
		fmt.Println("Error unmarshalling IPFS data:", err)
		return IPFSData{}
	}

	return ipfsData
}

// Function to log execution time to a CSV file
func LogExecutionTimeToCSVCardApproval(structSlice interface{}, elapsed, decryptionTime time.Duration, ipnsKey, functionName string, check string) {
	// Define the CSV file path
	csvFilePath := "csv/teeExecution_cardApproval_10runs.csv"

	// Check if the file exists
	fileExists := false
	if _, err := os.Stat(csvFilePath); err == nil {
		fileExists = true
	}

	// Ensure structSlice is a slice
	v := reflect.ValueOf(structSlice)
	if v.Kind() != reflect.Slice {
		fmt.Println("Error: structSlice is not a slice")
		return
	}

	// Get number of users (n_users)
	nUsers := strconv.Itoa(v.Len()) // Length of the slice

	// Convert times to milliseconds with decimal precision
	decryptionTimeMS := float64(decryptionTime.Milliseconds()) + float64(decryptionTime.Nanoseconds()%1e6)/1e6
	decisionTimeMS := float64(elapsed.Milliseconds()) + float64(elapsed.Nanoseconds()%1e6)/1e6
	totalTimeMS := decryptionTimeMS + decisionTimeMS // Compute total time

	// Format time values with 6 decimal places
	decryptionTimeStr := strconv.FormatFloat(decryptionTimeMS, 'f', 6, 64)
	decisionTimeStr := strconv.FormatFloat(decisionTimeMS, 'f', 6, 64)
	totalTimeStr := strconv.FormatFloat(totalTimeMS, 'f', 6, 64)

	// Open the file in append mode, create if not exists
	file, err := os.OpenFile(csvFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening CSV file: %v\n", err)
		return
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if the file is newly created
	if !fileExists {
		header := []string{"n_users", "Decryption_time", "Decision_time", "Total_time", "Check"}
		if err := writer.Write(header); err != nil {
			fmt.Printf("Error writing header to CSV: %v\n", err)
			return
		}
	}

	// Write the execution time record
	record := []string{
		nUsers,
		decryptionTimeStr, // Decryption time in milliseconds
		decisionTimeStr,   // Decision time (elapsed) in milliseconds
		totalTimeStr,      // Sum of decryption + decision time
		check,
	}
	if err := writer.Write(record); err != nil {
		fmt.Printf("Error writing record to CSV: %v\n", err)
		return
	}

	fmt.Println("Execution time logged to CSV successfully.")
}

// Function to log execution time to a CSV file
func LogExecutionTimeToCSV(structSlice interface{}, elapsed, decryptionTime time.Duration, ipnsKey, functionName string, check string) {
	// Define the CSV file path
	csvFilePath := "csv/teeExecution_decision_card_approval_10runs.csv"

	// Check if the file exists
	fileExists := false
	if _, err := os.Stat(csvFilePath); err == nil {
		fileExists = true
	}

	// Ensure structSlice is a slice
	v := reflect.ValueOf(structSlice)
	if v.Kind() != reflect.Slice {
		fmt.Println("Error: structSlice is not a slice")
		return
	}

	// Get number of users (n_users)
	nUsers := strconv.Itoa(v.Len()) // Length of the slice

	// Extract n_cols from ipnsKey
	var nCols string
	parts := strings.Split(ipnsKey, "_")
	numColumns := strings.TrimSuffix(parts[1], "Light")
	if len(parts) >= 2 {
		nCols = numColumns
	} else {
		fmt.Println("Error: Invalid IPNS Key format")
		return
	}

	// Extract n_rules from functionName
	// var nRules string
	// rulesParts := strings.Split(functionName, "Rules")
	// if len(rulesParts) >= 2 {
	// 	nRules = rulesParts[1]
	// } else {
	// 	fmt.Println("Error: Invalid Function Name format")
	// 	return
	// }

	// Convert times to milliseconds with decimal precision
	decryptionTimeMS := float64(decryptionTime.Milliseconds()) + float64(decryptionTime.Nanoseconds()%1e6)/1e6
	decisionTimeMS := float64(elapsed.Milliseconds()) + float64(elapsed.Nanoseconds()%1e6)/1e6
	totalTimeMS := decryptionTimeMS + decisionTimeMS // Compute total time

	// Format time values with 6 decimal places
	decryptionTimeStr := strconv.FormatFloat(decryptionTimeMS, 'f', 6, 64)
	decisionTimeStr := strconv.FormatFloat(decisionTimeMS, 'f', 6, 64)
	totalTimeStr := strconv.FormatFloat(totalTimeMS, 'f', 6, 64)

	// Open the file in append mode, create if not exists
	file, err := os.OpenFile(csvFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening CSV file: %v\n", err)
		return
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if the file is newly created
	if !fileExists {
		header := []string{"n_cols", "n_users", "Decryption_time", "Decision_time", "Total_time", "Check"}
		if err := writer.Write(header); err != nil {
			fmt.Printf("Error writing header to CSV: %v\n", err)
			return
		}
	}

	// Write the execution time record
	record := []string{
		// nRules,
		nCols,
		nUsers,
		decryptionTimeStr, // Decryption time in milliseconds
		decisionTimeStr,   // Decision time (elapsed) in milliseconds
		totalTimeStr,      // Sum of decryption + decision time
		check,
	}
	if err := writer.Write(record); err != nil {
		fmt.Printf("Error writing record to CSV: %v\n", err)
		return
	}

	fmt.Println("Execution time logged to CSV successfully.")
}

// Function to log execution time to a CSV file
func LogExecutionTimeToCSVAggregation(structSlice interface{}, numKeys int, elapsed, decryptionTime, aggregationTime time.Duration, ipnsKey string, check string) {
	// Define the CSV file path
	csvFilePath := "csv/heavyEncryption_decision_with_aggregation_10runs.csv"

	// Check if the file exists
	fileExists := false
	if _, err := os.Stat(csvFilePath); err == nil {
		fileExists = true
	}

	// Ensure structSlice is a slice
	v := reflect.ValueOf(structSlice)
	if v.Kind() != reflect.Slice {
		fmt.Println("Error: structSlice is not a slice")
		return
	}

	// Get number of users (n_users)
	nUsers := strconv.Itoa(v.Len()) // Length of the slice

	nCols := strconv.Itoa(numKeys) // Number of columns

	// Convert times to milliseconds with decimal precision
	decryptionTimeMS := float64(decryptionTime.Milliseconds()) + float64(decryptionTime.Nanoseconds()%1e6)/1e6
	aggregationTimeMS := float64(aggregationTime.Milliseconds()) + float64(aggregationTime.Nanoseconds()%1e6)/1e6
	decisionTimeMS := float64(elapsed.Milliseconds()) + float64(elapsed.Nanoseconds()%1e6)/1e6
	totalTimeMS := decryptionTimeMS + aggregationTimeMS + decisionTimeMS // Compute total time

	// Format time values with 6 decimal places
	decryptionTimeStr := strconv.FormatFloat(decryptionTimeMS, 'f', 6, 64)
	aggregationTimeStr := strconv.FormatFloat(aggregationTimeMS, 'f', 6, 64)
	decisionTimeStr := strconv.FormatFloat(decisionTimeMS, 'f', 6, 64)
	totalTimeStr := strconv.FormatFloat(totalTimeMS, 'f', 6, 64)

	// Open the file in append mode, create if not exists
	file, err := os.OpenFile(csvFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening CSV file: %v\n", err)
		return
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if the file is newly created
	if !fileExists {
		header := []string{"n_cols", "n_users", "Decryption_time", "Aggregation_time", "Decision_time", "Total_time", "Check"}
		if err := writer.Write(header); err != nil {
			fmt.Printf("Error writing header to CSV: %v\n", err)
			return
		}
	}

	// Write the execution time record
	record := []string{
		nCols,
		nUsers,
		decryptionTimeStr,  // Decryption time in milliseconds
		aggregationTimeStr, // Aggregation time in milliseconds
		decisionTimeStr,    // Decision time (elapsed) in milliseconds
		totalTimeStr,       // Sum of decryption + decision time
		check,
	}
	if err := writer.Write(record); err != nil {
		fmt.Printf("Error writing record to CSV: %v\n", err)
		return
	}

	fmt.Println("Execution time logged to CSV successfully.")
}

// LogExecutionDetailsToCSV stores the size of the batch JSON and the parsed record count in a CSV file
func LogExecutionDetailsToCSV(batchJSONSize, parsedRecords int, ipnsKey string) {
	// Define the CSV file path
	csvFilePath := "csv/execution_lightEncryption.csv"

	// Check if the file exists
	fileExists := false
	if _, err := os.Stat(csvFilePath); err == nil {
		fileExists = true
	}

	// Open the file in append mode, create if not exists
	file, err := os.OpenFile(csvFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening CSV file: %v\n", err)
		return
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if the file is newly created
	if !fileExists {
		header := []string{"IPNS_Key", "Batch_JSON_Size", "Parsed_Records"}
		if err := writer.Write(header); err != nil {
			fmt.Printf("Error writing header to CSV: %v\n", err)
			return
		}
	}

	// Write the execution record
	record := []string{
		ipnsKey,
		strconv.Itoa(batchJSONSize), // Size of batch JSON
		strconv.Itoa(parsedRecords), // Number of parsed records
	}
	if err := writer.Write(record); err != nil {
		fmt.Printf("Error writing record to CSV: %v\n", err)
		return
	}

	fmt.Println("Execution details logged to CSV successfully.")
}

func CreateInputsFromDataInput(dataInput interface{}) (map[string]interface{}, error) {
	// Create a map to hold the inputs
	inputs := make(map[string]interface{})

	// Get the value and type of the struct
	v := reflect.ValueOf(dataInput)
	t := reflect.TypeOf(dataInput)

	// Ensure it's a struct (or pointer to a struct)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct, got %s", v.Kind())
	}

	// Loop through struct fields
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Use the JSON tag as the key, fallback to field name if no tag exists
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name // Default to struct field name

		if jsonTag != "" {
			fieldName = strings.Split(jsonTag, ",")[0] // Get the JSON tag (ignore omitempty, etc.)
		}

		// Handle field types
		switch fieldValue.Kind() {
		case reflect.Slice:
			// Handle slices (if it is a []string, take the first element as string)
			if fieldValue.Type().Elem().Kind() == reflect.String && fieldValue.Len() > 0 {
				inputs[fieldName] = fieldValue.Index(0).String()
			} else {
				inputs[fieldName] = ""
			}

		case reflect.Bool:
			inputs[fieldName] = fieldValue.Bool()

		case reflect.String:
			strVal := fieldValue.String()

			// Check if the string is a date (YYYY-MM-DD format)
			if parsedTime, err := parseDate(strVal); err == nil {
				inputs[fieldName] = parsedTime
			} else {
				// If not a valid date, store as string
				inputs[fieldName] = strVal
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			inputs[fieldName] = fieldValue.Int()

		case reflect.Float32, reflect.Float64:
			inputs[fieldName] = fieldValue.Float()

		default:
			inputs[fieldName] = fieldValue.Interface()
		}
	}

	return inputs, nil
}

// Helper function to parse date strings into time.Time
func parseDate(dateStr string) (time.Time, error) {
	// Try to parse as YYYY-MM-DD format
	layout := "2006-01-02"
	parsedTime, err := time.Parse(layout, dateStr)
	if err != nil {
		return time.Time{}, err
	}
	return parsedTime, nil
}

func MergeMaps(base map[string]interface{}, additional map[string]interface{}) map[string]interface{} {
	for key, value := range additional {
		base[key] = value
	}
	return base
}

func ParseFEELContext(input string) (string, error) {
	// Match context structure {key: value}
	contextRegex := regexp.MustCompile(`\{[^:]+:\s*(.+)\}`)
	matches := contextRegex.FindStringSubmatch(input)

	if len(matches) != 2 {
		return "", fmt.Errorf("invalid FEEL context format")
	}

	value := strings.TrimSpace(matches[1])

	filterCondition := processFilterCondition(value)

	return filterCondition, nil
}

func NewParseInput(input string) (string, string, error) {
	// Step 1: Split on the comma after `]`
	parts := strings.SplitN(input, "],", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid input format: missing `],` to split")
	}

	leftPart := parts[0] + "]" // Add back the `]` removed by SplitN
	rightPart := strings.TrimSpace(parts[1])

	// Step 2: Extract the filtering condition inside `[]` from the left part
	startIdx := strings.Index(leftPart, "[")
	endIdx := strings.LastIndex(leftPart, "]")
	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		return "", "", errors.New("invalid input format: missing `[` or `]` in left part")
	}

	filterCondition := strings.TrimSpace(leftPart[startIdx+1 : endIdx])

	// Step 3: Remove everything before and including the first `:` in the right part
	colonIdx := strings.Index(rightPart, ":")
	if colonIdx != -1 {
		rightPart = strings.TrimSpace(rightPart[colonIdx+1:])
	}

	// Step 4: Remove trailing `}` if present
	if strings.HasSuffix(rightPart, "}") {
		rightPart = strings.TrimSuffix(rightPart, "}")
	}

	return filterCondition, rightPart, nil
}

// GetFilterFunctions returns functions for filtering phase.
func getFilterFunctions() map[string]govaluate.ExpressionFunction {
	return map[string]govaluate.ExpressionFunction{
		"contains": func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("contains() expects exactly two arguments")
			}

			slice, ok := args[0].([]string)
			if !ok {
				return nil, fmt.Errorf("contains() first argument must be a slice of strings")
			}

			item, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("contains() second argument must be a string")
			}

			for _, v := range slice {
				if v == item {
					return true, nil
				}
			}
			return false, nil
		},
	}
}

// ProcessFilterCondition processes filter conditions for govaluate syntax.
func processFilterCondition(filterCondition string) string {
	singleEqualsRegex := regexp.MustCompile(`\s=\s`)
	filterCondition = singleEqualsRegex.ReplaceAllString(filterCondition, " == ")
	filterCondition = strings.ReplaceAll(filterCondition, " and ", " && ")
	filterCondition = strings.ReplaceAll(filterCondition, " or ", " || ")
	return filterCondition
}

// EvaluateFilterConditionForSlice filters elements from structSlice.
func EvaluateFilterCondition(filterCondition string, structSlice reflect.Value) ([]interface{}, error) {
	if structSlice.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input must be a slice")
	}

	// Process the filter condition for govaluate syntax
	filterCondition = processFilterCondition(filterCondition)
	// fmt.Println("Processed Filter Condition:", filterCondition)

	// Compile the filter condition using govaluate
	expression, err := govaluate.NewEvaluableExpressionWithFunctions(filterCondition, getFilterFunctions())
	if err != nil {
		return nil, fmt.Errorf("error compiling filter condition: %v", err)
	}

	filteredElements := []interface{}{}
	for i := 0; i < structSlice.Len(); i++ {
		elem := structSlice.Index(i).Interface()

		// Convert struct to map for govaluate
		parameters, err := structToMap(elem)
		if err != nil {
			return nil, fmt.Errorf("error converting struct to map: %v", err)
		}

		// Evaluate the filter condition
		result, err := expression.Evaluate(parameters)
		if err != nil {
			return nil, fmt.Errorf("error evaluating filter condition: %v", err)
		}
		// fmt.Println("Result:", result)

		if resultBool, ok := result.(bool); ok && resultBool {
			filteredElements = append(filteredElements, elem)
		}
	}
	// fmt.Println("Filtered Elements:", filteredElements)
	fmt.Println("Filtered Elements Length:", len(filteredElements))

	return filteredElements, nil
}

// structToMap converts a struct to a map for govaluate.
func structToMap(item interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	val := reflect.ValueOf(item)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("item must be a struct")
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i).Interface()
		result[field.Name] = fieldValue
	}

	return result, nil
}

func NewPerformAggregationWithDynamicField(filteredElements reflect.Value, formula string) (float64, error) {
	if filteredElements.Kind() != reflect.Slice || filteredElements.Len() == 0 {
		return 0, fmt.Errorf("invalid or empty struct slice provided")
	}

	// Define the environment dynamically for dynamic field access
	env := map[string]interface{}{
		"max": func(field string) float64 {
			maxVal := math.Inf(-1)
			for i := 0; i < filteredElements.Len(); i++ {
				elem := filteredElements.Index(i)
				if elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
					elem = elem.Elem()
				}
				val := elem.FieldByName(field)
				if !val.IsValid() {
					panic(fmt.Sprintf("Field '%s' does not exist", field))
				}
				maxVal = math.Max(maxVal, getFieldValue(val))
			}
			return maxVal
		},
		"min": func(field string) float64 {
			minVal := math.Inf(1)
			for i := 0; i < filteredElements.Len(); i++ {
				elem := filteredElements.Index(i)
				if elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
					elem = elem.Elem()
				}
				val := elem.FieldByName(field)
				if !val.IsValid() {
					panic(fmt.Sprintf("Field '%s' does not exist", field))
				}
				minVal = math.Min(minVal, getFieldValue(val))
			}
			return minVal
		},
		"mean": func(field string) float64 {
			total := 0.0
			count := 0
			for i := 0; i < filteredElements.Len(); i++ {
				elem := filteredElements.Index(i)
				if elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
					elem = elem.Elem()
				}
				val := elem.FieldByName(field)
				if !val.IsValid() {
					panic(fmt.Sprintf("Field '%s' does not exist", field))
				}
				total += getFieldValue(val)
				count++
			}
			if count == 0 {
				return 0
			}
			return total / float64(count)
		},
		"sum": func(field string) float64 { // New sum function
			total := 0.0
			for i := 0; i < filteredElements.Len(); i++ {
				elem := filteredElements.Index(i)
				if elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
					elem = elem.Elem()
				}
				val := elem.FieldByName(field)
				if !val.IsValid() {
					panic(fmt.Sprintf("Field '%s' does not exist", field))
				}
				total += getFieldValue(val)
			}
			return total
		},
		"count": func(field string) int {
			count := 0
			for i := 0; i < filteredElements.Len(); i++ {
				elem := filteredElements.Index(i)
				if elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
					elem = elem.Elem()
				}
				val := elem.FieldByName(field)
				if val.IsValid() {
					count++
				}
			}
			return count
		},
		"ceiling": func(value float64) float64 {
			return math.Ceil(value)
		},
		"abs": func(value float64) float64 {
			return math.Abs(value)
		},
	}

	// Extract field names from the formula
	fieldRegex := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*\.\s*([A-Za-z_][A-Za-z0-9_]*)`)
	formula = fieldRegex.ReplaceAllStringFunc(formula, func(match string) string {
		field := fieldRegex.FindStringSubmatch(match)[1]
		return fmt.Sprintf("'%s'", field)
	})

	// Compile the formula
	program, err := expr.Compile(formula, expr.Env(env))
	if err != nil {
		return 0, fmt.Errorf("error compiling formula: %w", err)
	}

	// Evaluate the formula
	result, err := expr.Run(program, env)
	if err != nil {
		return 0, fmt.Errorf("error running formula: %w", err)
	}

	// Return the result
	if res, ok := result.(float64); ok {
		return res, nil
	}
	return 0, fmt.Errorf("unexpected result type: %T", result)
}

func getFieldValue(val reflect.Value) float64 {
	switch val.Kind() {
	case reflect.Int:
		return float64(val.Int())
	case reflect.Float64:
		return val.Float()
	default:
		panic(fmt.Sprintf("Unsupported field type: %v", val.Kind()))
	}
}
