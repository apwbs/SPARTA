package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

type Rule struct {
	Name      string
	Action    string
	Target    string // kept (used later for decisionwithaggregation synthesis if you want)
	Condition string
}

type Policy struct {
	Name      string
	Target    string            // policy-level target (raw target clause)
	Resources map[string]string // name, type, struct
	Rules     []*Rule

	// Obligation params captured from `on permit { obligation X { k="v" } }`
	Obligations map[string]map[string]string // oblSym -> param -> value
}

type PolicySet struct {
	Namespace string
	Policies  []*Policy
}

const mandatoryImport = "import Oasis.Attributes.*"

func main() {
	filePath := "policy.alfa"

	// Parse
	policySet, err := parsePolicySet(filePath)
	if err != nil {
		fmt.Println("Error parsing policy set:", err)
		return
	}

	// // Pretty print parsed content
	// fmt.Println("========== Parsed ALFA ==========")
	// fmt.Printf("Namespace: %s\n", policySet.Namespace)
	// for _, p := range policySet.Policies {
	// 	fmt.Println("\n--------------------------------")
	// 	fmt.Printf("Policy: %s\n", p.Name)
	// 	fmt.Printf("  Target: %s\n", p.Target)

	// 	fmt.Printf("  Resources:\n")
	// 	for _, k := range []string{"name", "type", "struct"} {
	// 		if v, ok := p.Resources[k]; ok && v != "" {
	// 			fmt.Printf("    %s: %s\n", k, v)
	// 		}
	// 	}

	// 	fmt.Printf("  Rules (%d):\n", len(p.Rules))
	// 	for _, r := range p.Rules {
	// 		fmt.Printf("    - Rule: %s\n", r.Name)
	// 		if r.Action != "" {
	// 			fmt.Printf("      Action: %s\n", r.Action)
	// 		}
	// 		if strings.TrimSpace(r.Condition) != "" {
	// 			fmt.Printf("      Condition(raw):  %s\n", strings.TrimSpace(r.Condition))
	// 			fmt.Printf("      Condition(FEEL): %s\n", translateConditionToFEEL(strings.TrimSpace(r.Condition)))
	// 		}
	// 	}

	// 	if len(p.Obligations) > 0 {
	// 		fmt.Printf("  Obligations on permit:\n")
	// 		for obl, params := range p.Obligations {
	// 			fmt.Printf("    - %s:\n", obl)
	// 			for k, v := range params {
	// 				fmt.Printf("        %s = %q\n", k, v)
	// 			}
	// 		}
	// 	}
	// }

	// fmt.Println("\n========== Generating code ==========")

	receiverPath := "../teeserver/receiver/teeserver_receiver.go"
	desobjPath := "../teeserver/receiver/desobj.go"

	handleFn, desobjCode, err := GenerateHandleFunctionAndDesobj(policySet, receiverPath)
	if err != nil {
		fmt.Println("Error generating code:", err)
		return
	}

	// 1) Load existing teeserver_receiver.go
	receiverBytes, err := os.ReadFile(receiverPath)
	if err != nil {
		fmt.Println("Error reading receiver file:", err)
		return
	}
	receiverSrc := string(receiverBytes)

	// 2) Replace placeholder handleFunction
	updatedReceiver, err := ReplaceHandleFunction(receiverSrc, handleFn)
	if err != nil {
		fmt.Println("Error replacing handleFunction:", err)
		return
	}

	// 3) Write back teeserver_receiver.go
	if err := os.WriteFile(receiverPath, []byte(updatedReceiver), 0644); err != nil {
		fmt.Println("Error writing updated receiver file:", err)
		return
	}

	// 4) Write desobj.go
	if err := os.WriteFile(desobjPath, []byte(desobjCode), 0644); err != nil {
		fmt.Println("Error writing desobj.go:", err)
		return
	}

	fmt.Println("Updated:", receiverPath)
	fmt.Println("Generated:", desobjPath)
}

func ReplaceHandleFunction(receiverSrc, newHandleFunction string) (string, error) {
	sig := "func handleFunction(functionName string, payload map[string]string) string"
	start := strings.Index(receiverSrc, sig)
	if start == -1 {
		return "", fmt.Errorf("handleFunction signature not found")
	}

	// Find the first '{' after the signature
	open := strings.Index(receiverSrc[start:], "{")
	if open == -1 {
		return "", fmt.Errorf("handleFunction opening brace not found")
	}
	open += start

	// Walk forward counting braces to find the matching closing '}'
	depth := 0
	inString := false
	inRawString := false
	escape := false

	for i := open; i < len(receiverSrc); i++ {
		ch := receiverSrc[i]

		// Handle raw string literals: `...`
		if !inString && ch == '`' {
			inRawString = !inRawString
			continue
		}
		if inRawString {
			continue
		}

		// Handle normal string literals: "..."
		if !inRawString && ch == '"' && !escape {
			inString = !inString
			continue
		}
		if inString {
			if ch == '\\' && !escape {
				escape = true
			} else {
				escape = false
			}
			continue
		}

		// Not in any string: count braces
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				// i is the matching closing brace of the function
				end := i + 1
				return receiverSrc[:start] + newHandleFunction + receiverSrc[end:], nil
			}
		}
	}

	return "", fmt.Errorf("unterminated handleFunction block (could not find matching brace)")
}

// -------------------------
// CLEAN parser for the “new ALFA” format only
// -------------------------
func parsePolicySet(filePath string) (PolicySet, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return PolicySet{}, err
	}
	defer file.Close()

	var ps PolicySet

	// Required
	importSeen := false

	// Named conditions: condition NAME <expr>
	namedConditions := make(map[string]string)

	// Obligation validation: category sym -> URI, obligation sym -> URI, attribute name -> category sym
	categories := make(map[string]string)  // AggregationSpecCat -> VaccineDispatch.AggregationSpec
	obligations := make(map[string]string) // AggregationSpec -> VaccineDispatch.AggregationSpec
	attributes := make(map[string]string)  // meanAge -> AggregationSpecCat (or environmentCat)

	// State
	var currentPolicy *Policy
	var currentRule *Rule

	inPolicy := false
	inRule := false
	inOnPermit := false
	inObligationBlock := false
	currentOblSym := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// -------------------------
		// Namespace-level
		// -------------------------
		if !inPolicy && !inRule && !inOnPermit {
			if strings.HasPrefix(line, "namespace ") {
				ns := strings.TrimSpace(strings.TrimPrefix(line, "namespace "))
				ns = strings.TrimSuffix(ns, "{")
				ps.Namespace = strings.TrimSpace(ns)
				continue
			}

			if strings.HasPrefix(line, "import ") {
				if strings.TrimSpace(line) == mandatoryImport {
					importSeen = true
				}
				continue
			}

			if strings.HasPrefix(line, "attribute ") {
				attrName, catSym, e := parseAttributeNameAndCategory(line)
				if e != nil {
					return PolicySet{}, e
				}
				attributes[attrName] = catSym
				continue
			}

			if strings.HasPrefix(line, "condition ") {
				name, expr := parseNamedCondition(line)
				if name != "" {
					namedConditions[name] = expr
				}
				continue
			}

			if strings.HasPrefix(line, "category ") {
				sym, uri, e := parseSymbolEqualsString(line, "category")
				if e != nil {
					return PolicySet{}, e
				}
				categories[sym] = uri
				continue
			}

			if strings.HasPrefix(line, "obligation ") {
				sym, uri, e := parseSymbolEqualsString(line, "obligation")
				if e != nil {
					return PolicySet{}, e
				}
				obligations[sym] = uri
				continue
			}
		}

		// -------------------------
		// Policy start
		// -------------------------
		if strings.HasPrefix(line, "policy ") {
			inPolicy = true
			inRule = false
			inOnPermit = false
			inObligationBlock = false
			currentOblSym = ""

			pname := strings.TrimSpace(strings.TrimPrefix(line, "policy "))
			pname = strings.TrimSuffix(pname, "{")
			pname = strings.TrimSpace(pname)

			currentPolicy = &Policy{
				Name:        pname,
				Resources:   make(map[string]string),
				Obligations: make(map[string]map[string]string),
			}
			ps.Policies = append(ps.Policies, currentPolicy)
			continue
		}

		// -------------------------
		// Policy-level
		// -------------------------
		if inPolicy && !inRule && !inOnPermit {
			if strings.HasPrefix(line, "target clause") {
				target := strings.TrimSpace(strings.TrimPrefix(line, "target clause "))
				currentPolicy.Target = target

				// only support Action=="..."
				actName, actArgs, ok := parseActionFromTargetClause(target)
				if !ok {
					return PolicySet{}, fmt.Errorf("policy %s: missing or invalid Action target: %s", currentPolicy.Name, target)
				}
				currentPolicy.Resources["name"] = actName
				if len(actArgs) >= 1 {
					currentPolicy.Resources["struct"] = actArgs[0]
				}
				continue
			}

			if strings.HasPrefix(line, "apply ") {
				// ignore (denyUnlessPermit etc.)
				continue
			}

			if strings.HasPrefix(line, "rule ") {
				inRule = true
				rname := strings.TrimSpace(strings.TrimPrefix(line, "rule "))
				rname = strings.TrimSuffix(rname, "{")
				rname = strings.TrimSpace(rname)

				currentRule = &Rule{Name: rname}
				currentPolicy.Rules = append(currentPolicy.Rules, currentRule)

				// type comes from rule name
				if t := typeFromRuleName(rname); t != "" {
					currentPolicy.Resources["type"] = t
				}
				continue
			}

			if strings.HasPrefix(line, "on permit") {
				inOnPermit = true
				continue
			}

			// end policy block
			if line == "}" {
				inPolicy = false
				currentPolicy = nil
				continue
			}
		}

		// -------------------------
		// Rule body
		// -------------------------
		if inRule {
			if strings.HasPrefix(line, "permit") {
				currentRule.Action = "permit"
				continue
			}
			if strings.HasPrefix(line, "deny") {
				currentRule.Action = "deny"
				continue
			}

			if strings.HasPrefix(line, "condition ") {
				cond := strings.TrimSpace(strings.TrimPrefix(line, "condition "))
				cond = strings.TrimSuffix(cond, ";")
				cond = expandCondition(cond, namedConditions)
				currentRule.Condition += cond + " "
				continue
			}

			if line == "}" {
				inRule = false
				currentRule = nil
				continue
			}
		}

		// -------------------------
		// on permit { obligation ... { ... } }
		// -------------------------
		if inOnPermit {
			if !inObligationBlock && strings.HasPrefix(line, "obligation ") && strings.HasSuffix(line, "{") {
				inObligationBlock = true
				currentOblSym = strings.TrimSpace(strings.TrimPrefix(line, "obligation "))
				currentOblSym = strings.TrimSuffix(currentOblSym, "{")
				currentOblSym = strings.TrimSpace(currentOblSym)

				if _, ok := currentPolicy.Obligations[currentOblSym]; !ok {
					currentPolicy.Obligations[currentOblSym] = make(map[string]string)
				}
				continue
			}

			if inObligationBlock && strings.Contains(line, "=") {
				k, v, e := parseAssignmentString(line)
				if e != nil {
					return PolicySet{}, e
				}
				currentPolicy.Obligations[currentOblSym][k] = v
				continue
			}

			if inObligationBlock && line == "}" {
				inObligationBlock = false
				currentOblSym = ""
				continue
			}

			if !inObligationBlock && line == "}" {
				inOnPermit = false
				continue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return PolicySet{}, err
	}

	// -------------------------
	// Checks
	// -------------------------
	if ps.Namespace == "" {
		return PolicySet{}, fmt.Errorf("missing namespace")
	}
	if !importSeen {
		return PolicySet{}, fmt.Errorf("missing mandatory import: %s", mandatoryImport)
	}

	// attribute categories must exist (except environmentCat)
	for attr, catSym := range attributes {
		if catSym == "environmentCat" {
			continue
		}
		if _, ok := categories[catSym]; !ok {
			return PolicySet{}, fmt.Errorf("attribute %s references undefined category: %s", attr, catSym)
		}
	}

	// obligation blocks: obligation must exist, param attributes must exist and match obligation URI via category URI
	for _, pol := range ps.Policies {
		for oblSym, params := range pol.Obligations {
			oblURI, ok := obligations[oblSym]
			if !ok {
				return PolicySet{}, fmt.Errorf("policy %s uses undefined obligation: %s", pol.Name, oblSym)
			}

			allowedCats := map[string]bool{}
			for catSym, catURI := range categories {
				if catURI == oblURI {
					allowedCats[catSym] = true
				}
			}
			if len(allowedCats) == 0 {
				return PolicySet{}, fmt.Errorf("obligation %s (%s) has no category with same URI", oblSym, oblURI)
			}

			for param := range params {
				paramCat, ok := attributes[param]
				if !ok {
					return PolicySet{}, fmt.Errorf("policy %s obligation %s assigns undeclared attribute: %s", pol.Name, oblSym, param)
				}
				if !allowedCats[paramCat] {
					return PolicySet{}, fmt.Errorf("policy %s obligation %s param %s category %s does not match obligation URI %s",
						pol.Name, oblSym, param, paramCat, oblURI)
				}
			}
		}
	}

	return ps, nil
}

// -------------------------
// Helpers
// -------------------------

func typeFromRuleName(ruleName string) string {
	n := strings.ToLower(strings.TrimSpace(ruleName))
	if strings.HasPrefix(n, "access") {
		n = strings.TrimPrefix(n, "access")
	}

	// Explicit "with aggregation" marker (what you expect in the ALFA file)
	if strings.Contains(n, "decisionwithaggregation") ||
		strings.Contains(n, "decisionwithaggr") ||
		strings.Contains(n, "withaggregation") ||
		strings.Contains(n, "withaggr") {
		return "decisionwithaggregation"
	}

	switch {
	case strings.HasPrefix(n, "write"):
		return "write"
	case strings.HasPrefix(n, "read"):
		return "read"
	case strings.HasPrefix(n, "decision"):
		return "decision"
	default:
		return ""
	}
}

func translateConditionToFEEL(condition string) string {
	cond := strings.TrimSpace(condition)
	cond = strings.TrimSuffix(cond, ";")
	cond = strings.ReplaceAll(cond, "==", " = ")

	for strings.Contains(cond, "  ") {
		cond = strings.ReplaceAll(cond, "  ", " ")
	}
	cond = strings.TrimSpace(cond)

	if (strings.Contains(cond, " and ") || strings.Contains(cond, " or ")) &&
		!(strings.HasPrefix(cond, "(") && strings.HasSuffix(cond, ")")) {
		cond = "(" + cond + ")"
	}
	return cond
}

func parseNamedCondition(line string) (name, expr string) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "condition "))
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return "", ""
	}
	name = parts[0]
	expr = strings.TrimSpace(rest[len(name):])
	return name, expr
}

func expandCondition(cond string, named map[string]string) string {
	if expr, ok := named[cond]; ok {
		return expr
	}
	return cond
}

func parseSymbolEqualsString(line, keyword string) (sym, uri string, err error) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, keyword))
	rest = strings.TrimSpace(rest)

	eq := strings.Index(rest, "=")
	if eq == -1 {
		return "", "", fmt.Errorf("%s missing '=': %s", keyword, line)
	}
	sym = strings.TrimSpace(rest[:eq])
	rhs := strings.TrimSpace(rest[eq+1:])
	uri = extractQuotedValue(rhs)
	if sym == "" || uri == "" {
		return "", "", fmt.Errorf("invalid %s declaration: %s", keyword, line)
	}
	return sym, uri, nil
}

func parseAttributeNameAndCategory(line string) (attrName, catSym string, err error) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "attribute "))
	nameEnd := strings.Index(rest, "{")
	if nameEnd == -1 {
		return "", "", fmt.Errorf("invalid attribute (missing '{'): %s", line)
	}
	attrName = strings.TrimSpace(rest[:nameEnd])
	body := strings.TrimSpace(rest[nameEnd+1:])
	body = strings.TrimSuffix(body, "}")
	body = strings.TrimSpace(body)

	catSym = extractKeyToken(body, "category=")
	if attrName == "" || catSym == "" {
		return "", "", fmt.Errorf("invalid attribute fields: %s", line)
	}
	return attrName, catSym, nil
}

func extractKeyToken(body, key string) string {
	start := strings.Index(body, key)
	if start == -1 {
		return ""
	}
	start += len(key)
	rest := body[start:]
	end := strings.IndexAny(rest, " \t")
	if end == -1 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}

func parseAssignmentString(line string) (key, value string, err error) {
	eq := strings.Index(line, "=")
	if eq == -1 {
		return "", "", fmt.Errorf("invalid assignment (no '='): %s", line)
	}
	key = strings.TrimSpace(line[:eq])
	rhs := strings.TrimSpace(line[eq+1:])
	value = extractQuotedValue(rhs)
	if key == "" || value == "" {
		return "", "", fmt.Errorf("invalid assignment (expect quoted string): %s", line)
	}
	return key, value, nil
}

func parseActionFromTargetClause(target string) (name string, args []string, ok bool) {
	if !strings.Contains(target, "Action==") {
		return "", nil, false
	}
	raw := extractQuotedValue(target)
	if raw == "" {
		return "", nil, false
	}
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	if open == -1 || close == -1 || close < open {
		return strings.TrimSpace(raw), nil, true
	}
	name = strings.TrimSpace(raw[:open])
	argsRaw := raw[open+1 : close]
	parts := strings.Split(argsRaw, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			args = append(args, p)
		}
	}
	return name, args, true
}

func extractQuotedValue(input string) string {
	start := strings.Index(input, "\"")
	if start == -1 {
		return ""
	}
	endRel := strings.Index(input[start+1:], "\"")
	if endRel == -1 {
		return ""
	}
	end := start + 1 + endRel
	return input[start+1 : end]
}

// -------------------------
// Generation: handleFunction replacement + desobj.go
// -------------------------

func GenerateHandleFunctionAndDesobj(policySet PolicySet, receiverPath string) (handleFn string, desobjGo string, err error) {
	receiverBytes, err := os.ReadFile(receiverPath)
	if err != nil {
		return "", "", err
	}
	pkg := parsePackageName(string(receiverBytes))
	if pkg == "" {
		return "", "", fmt.Errorf("could not determine package name from %s", receiverPath)
	}

	var functions string
	var switchCases string

	addedCase := make(map[string]bool)
	addSwitchCase := func(caseKey, handlerFn string) {
		if caseKey == "" || handlerFn == "" || addedCase[caseKey] {
			return
		}
		addedCase[caseKey] = true
		switchCases += fmt.Sprintf(`
	case "%s":
		return %s(payload)`, caseKey, handlerFn)
	}

	for _, policy := range policySet.Policies {
		for _, rule := range policy.Rules {
			resourceName := policy.Resources["name"]
			resourceType := strings.ToLower(policy.Resources["type"])
			resourceStruct := policy.Resources["struct"]

			logicalFormula := translateConditionToFEEL(rule.Condition)

			// WRITE
			if resourceType == "write" {
				caseKey := resourceName
				handlerFn := resourceName + "Handler"

				functions += fmt.Sprintf(`
func %s(payload map[string]string) string {
	certificate, _, fileBytes, _, ipnsKey, _ := interfaceISGoMiddleware.ParseSetRequestFromQueueBytes(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`+"`{accessPolicy: %s}`"+`, attributes)
		if callable {
			interfaceISGoMiddleware.EncryptAndUploadLinkedBytes(fileBytes, %q, ipnsKey)
			interfaceISGoMiddleware.EncryptAndUploadLinkedBytes(fileBytes, %q, ipnsKey+"Light")
			return "Encryption of document performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}
`, handlerFn, logicalFormula, resourceStruct, resourceStruct+"Light")

				addSwitchCase(caseKey, handlerFn)
				continue
			}

			// DECISION WITH AGGREGATION (rule name marker: accessDecisionWithAggregation)
			if resourceType == "decisionwithaggregation" {
				caseKey := resourceName
				handlerFn := resourceName + "Handler"

				_, oblParams, ok := pickAggregationObligation(policy)
				if !ok {
					return "", "", fmt.Errorf("policy %s: decisionwithaggregation but no obligations found", policy.Name)
				}

				// Determine base struct for aggregation:
				// from Action=="X(BaseStruct, ...)" -> base struct is first arg
				// You already store currentPolicy.Resources["struct"] from actArgs[0]
				baseStruct := resourceStruct

				// Decision struct for decision step:
				decisionStruct := resourceStruct + "Light"

				// Deterministic param order
				paramNames := make([]string, 0, len(oblParams))
				for k := range oblParams {
					paramNames = append(paramNames, k)
				}
				sort.Strings(paramNames)

				var aggrLines strings.Builder
				for _, param := range paramNames {
					spec := oblParams[param]
					aggrLines.WriteString(fmt.Sprintf(`
			aggrResult_%s, _, _ := interfaceISGoMiddleware.NewPerformAggregation(%q, structSlice)
			Additionals[%q] = aggrResult_%s`, param, spec, param, param))
				}

				functions += fmt.Sprintf(`
func %s(payload map[string]string) string {
	certificate, functionName, ipnsKey, _ := interfaceISGoMiddleware.ParseDecisionRequestFromQueue(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`+"`{accessPolicy: %s}`"+`, attributes)
		if callable {
			Additionals := make(map[string]interface{})
			structSliceInterface, _, _ := interfaceISGoMiddleware.RetrieveStructSliceLinkedLog(%q, ipnsKey)
			structSlice, ok := structSliceInterface.(reflect.Value)
			if !ok {
				structSlice = reflect.ValueOf(structSliceInterface)
				if structSlice.Kind() != reflect.Slice {
					return "Error: Retrieved data is not a slice"
				}
			}
			%s
			structSliceInterfaceForDecision, _, _ := interfaceISGoMiddleware.RetrieveStructSliceLinkedLog(%q, ipnsKey+"Light")
			structSliceForDecision, okForDecision := structSliceInterfaceForDecision.(reflect.Value)
			if !okForDecision {
				structSliceForDecision = reflect.ValueOf(structSliceInterfaceForDecision)
				if structSliceForDecision.Kind() != reflect.Slice {
					return "Error: Retrieved data is not a slice"
				}
			}
			interfaceISGoMiddleware.DecisionWithAggregation(functionName, structSliceForDecision, Additionals, _, _, ipnsKey)
			return "Decision performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}
`, handlerFn,
					logicalFormula,
					baseStruct,
					strings.TrimSpace(aggrLines.String()),
					decisionStruct,
					)

				addSwitchCase(caseKey, handlerFn)
				continue
			}

			// DECISION (normal)
			if resourceType == "decision" {
				caseKey := resourceName
				handlerFn := resourceName + "Handler"

				functions += fmt.Sprintf(`
func %s(payload map[string]string) string {
	certificate, functionName, ipnsKey, _ := interfaceISGoMiddleware.ParseDecisionRequestFromQueue(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`+"`{accessPolicy: %s}`"+`, attributes)
		if callable {
			interfaceISGoMiddleware.Decision(functionName, %q, ipnsKey+"Light")
			return "Decision performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}
`, handlerFn,
					logicalFormula,
					resourceStruct+"Light")

				addSwitchCase(caseKey, handlerFn)
				continue
			}
		}
	}

	handleFn = fmt.Sprintf(`func handleFunction(functionName string, payload map[string]string) string {
	switch functionName {%s
	default:
		return "Unknown function: " + functionName
	}
}`, switchCases)

	desobjGo = fmt.Sprintf(`package %s

import (
	"reflect"
	"sparta/src/utils/interfaceISGoMiddleware"
)

%s
`, pkg, strings.TrimSpace(functions))

	return handleFn, desobjGo, nil
}

func parsePackageName(src string) string {
	re := regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z_][A-Za-z0-9_]*)\s*$`)
	m := re.FindStringSubmatch(src)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

// Prefer obligation named "AggregationSpec" if present, otherwise pick the first one.
func pickAggregationObligation(pol *Policy) (oblName string, params map[string]string, ok bool) {
	if pol.Obligations == nil || len(pol.Obligations) == 0 {
		return "", nil, false
	}
	if p, exists := pol.Obligations["AggregationSpec"]; exists {
		return "AggregationSpec", p, true
	}
	// deterministic pick: sorted by name
	names := make([]string, 0, len(pol.Obligations))
	for k := range pol.Obligations {
		names = append(names, k)
	}
	sort.Strings(names)
	n := names[0]
	return n, pol.Obligations[n], true
}

// deterministic Go map literal
func goStringMapLiteral(m map[string]string) string {
	if len(m) == 0 {
		return "map[string]string{}"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("map[string]string{\n")
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("\t%q: %q,\n", k, m[k]))
	}
	sb.WriteString("}")
	return sb.String()
}
