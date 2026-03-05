package main

import (
	"bufio"
	"fmt"
	"os"
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
	// We keep them here for later (decisionwithaggregation synthesis, etc.)
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

	// Pretty print parsed content
	fmt.Println("========== Parsed ALFA ==========")
	fmt.Printf("Namespace: %s\n", policySet.Namespace)
	for _, p := range policySet.Policies {
		fmt.Println("\n--------------------------------")
		fmt.Printf("Policy: %s\n", p.Name)
		fmt.Printf("  Target: %s\n", p.Target)

		fmt.Printf("  Resources:\n")
		for _, k := range []string{"name", "type", "struct"} {
			if v, ok := p.Resources[k]; ok && v != "" {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}

		fmt.Printf("  Rules (%d):\n", len(p.Rules))
		for _, r := range p.Rules {
			fmt.Printf("    - Rule: %s\n", r.Name)
			if r.Action != "" {
				fmt.Printf("      Action: %s\n", r.Action)
			}
			if strings.TrimSpace(r.Condition) != "" {
				fmt.Printf("      Condition(raw):  %s\n", strings.TrimSpace(r.Condition))
				fmt.Printf("      Condition(FEEL): %s\n", translateConditionToFEEL(strings.TrimSpace(r.Condition)))
			}
		}

		if len(p.Obligations) > 0 {
			fmt.Printf("  Obligations on permit:\n")
			for obl, params := range p.Obligations {
				fmt.Printf("    - %s:\n", obl)
				for k, v := range params {
					fmt.Printf("        %s = %q\n", k, v)
				}
			}
		}
	}

	fmt.Println("\n========== Generating code ==========")

	// Generate handlers + server scaffold + handleFunction
	code := GenerateHandlers(policySet)

	// Write output
	outputFilePath := "file.go"
	outFile, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer outFile.Close()

	if _, err := outFile.WriteString(code); err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Println("Generated code has been written to", outputFilePath)
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

				// type comes from rule name (accessWrite/accessDecision/...)
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
	switch {
	case strings.HasPrefix(n, "write"):
		return "write"
	case strings.HasPrefix(n, "read"):
		return "read"
	case strings.HasPrefix(n, "decisionwithaggr"), strings.HasPrefix(n, "decisionwithaggregation"), strings.HasPrefix(n, "withaggr"):
		return "decisionwithaggregation"
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


func GenerateHandlers(policySet PolicySet) string {
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
			resourceType := policy.Resources["type"]
			resourceStruct := policy.Resources["struct"]

			logicalFormula := translateConditionToFEEL(rule.Condition)

			// WRITE (queue)
			if strings.ToLower(resourceType) == "write" {
				caseKey := "set" + resourceName
				handlerFn := "set" + resourceName + "Handler"

				functions += fmt.Sprintf(`
func %s(payload map[string]string) string {
	certificate, functionName, fileBytes, _, ipnsKey, _ := interfaceISGoMiddleware.ParseSetRequestFromQueueBytes(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`+"`{accessPolicy: %s}`"+`, attributes)
		if callable {
			interfaceISGoMiddleware.EncryptAndUploadLinkedBytes(fileBytes, "%s", ipnsKey)
			interfaceISGoMiddleware.EncryptAndUploadLinkedBytes(fileBytes, "%sLight", ipnsKey+"Light")
			return "Encryption of document performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}
				`, handlerFn, logicalFormula, resourceStruct, resourceStruct)

				addSwitchCase(caseKey, handlerFn)
				continue
			}

			// DECISION (queue) — includes ipnsKey
			if strings.ToLower(resourceType) == "decision" {
				caseKey := resourceName
				handlerFn := "decide" + resourceName + "Handler"

				functions += fmt.Sprintf(`
func %s(payload map[string]string) string {
	certificate, functionName, ipnsKey, _ := interfaceISGoMiddleware.ParseDecisionRequestFromQueue(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`+"`{accessPolicy: %s}`"+`, attributes)
		if callable {
			interfaceISGoMiddleware.Decision(functionName, "%sLight", ipnsKey+"Light")
			return "Decision performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}
				`, handlerFn, logicalFormula, resourceStruct)

				addSwitchCase(caseKey, handlerFn)
				continue
			}

			// decisionwithaggregation will be handled later using policy.Obligations
			_ = rule
		}
	}

	handleFunction := fmt.Sprintf(`
func handleFunction(functionName string, payload map[string]string) string {
	switch functionName {%s
	default:
		return "Unknown function: " + functionName
	}
}
`, switchCases)

	// ------------------------------------------------------------
	// NEW SERVER SCAFFOLD: Redis queue + /secret seed exchange gate
	// ------------------------------------------------------------
	serverScaffold := fmt.Sprintf(`
const (
	requestQueue  = "request_queue"
	responseQueue = "response_queue"
)

// Global Redis client
var redisClient *redis.Client

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Redis server address
	})
}

type USGonServerReceiver struct {
	server *http.Server
}

var (
	teeServerMeasurement string = ""
	seedExchangeEnabled  bool   = false
)


// StartUSGonServer now supports BOTH modes:
//
// - exchangeSeed=true  => BOOTSTRAP mode:
//     * uses CreateBootstrapCertificate() (no seed needed)
//     * exposes /secret endpoint (to receive the seed)
//     * DOES NOT start readQueue() because normal workload assumes the deterministic cert/seed exists
//
// - exchangeSeed=false => NORMAL mode:
//     * uses CreateCertificate() (seed required)
//     * /secret is disabled
//     * starts readQueue()

// StartTEE creates the enclave HTTPS server.
// - exchangeSeed=true  => BOOTSTRAP cert + /secret enabled
// - exchangeSeed=false => NORMAL deterministic cert + /secret disabled
func StartTEE(exchangeSeed bool) *USGonServerReceiver {
	seedExchangeEnabled = exchangeSeed

	var cert []byte
	var priv interface{}

	if seedExchangeEnabled {
		cert, priv = interfaceISGoMiddleware.CreateBootstrapCertificate()
		fmt.Println("Using BOOTSTRAP certificate (seed not required).")
	} else {
		cert, priv = interfaceISGoMiddleware.CreateCertificate()
		fmt.Println("Using NORMAL deterministic certificate (seed required).")
	}

	if cert == nil || priv == nil {
		fmt.Println("Error: certificate creation failed")
		os.Exit(1)
	}

	hash := sha256.Sum256(cert)
	s := &USGonServerReceiver{}

	report, err := enclave.GetRemoteReport(hash[:])
	if err != nil {
		fmt.Println(err)
	}

	// Read CA certificate for attestation
	caCert, err := os.ReadFile("certificate/user_cert.pem")
	if err != nil {
		fmt.Println("Error reading CA certificate:", err)
		os.Exit(1)
	}

	// HTTP handlers
	handler := http.NewServeMux()

	// Remote attestation endpoints
	handler.HandleFunc("/caCert", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Sending CA certificate")
		_, _ = w.Write(caCert)
	})
	handler.HandleFunc("/cert", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Sending RA certificate")
		_, _ = w.Write(cert)
	})
	handler.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Sending report")
		_, _ = w.Write(report)
	})

	// Seed exchange endpoint (gated inside handleKey)
	handler.HandleFunc("/secret", handleKey)

	tlsCfg := tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert},
				PrivateKey:  priv,
			},
		},
	}

	// Start processing requests from middleware ONLY in normal mode
	if !seedExchangeEnabled {
		go readQueue()
	} else {
		fmt.Println("BOOTSTRAP mode: readQueue() NOT started (waiting for /secret).")
	}

	s.server = &http.Server{
		Addr:      "0.0.0.0:8075",
		TLSConfig: &tlsCfg,
		Handler:   handler,
	}
	return s
}

// Start sets the expected peer measurement (used by VerifyTEE) and starts HTTPS server.
func (s *USGonServerReceiver) Start(measurement string) error {
	fmt.Println("TEE Server Receiver started")
	teeServerMeasurement = measurement
	return s.server.ListenAndServeTLS("", "")
}

// Graceful stop (used after seed exchange, if you want to exit immediately).
func (s *USGonServerReceiver) Stop() {
	if s == nil || s.server == nil {
		return
	}
	_ = s.server.Close()
}

// -------------------------
// Seed exchange receiver
// -------------------------
func handleKey(w http.ResponseWriter, r *http.Request) {
	// Gate seed exchange endpoint
	if !seedExchangeEnabled {
		http.Error(w, "Seed exchange disabled (run with -exchange_seed)", http.StatusForbidden)
		return
	}

	// Verify the peer TEE
	valid := teeRequester.VerifyTEE(teeServerMeasurement, false)
	if !valid {
		http.Error(w, "TEE verification failed", http.StatusUnauthorized)
		return
	}

	// Validate Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		fmt.Println("Unsupported Content-Type:", contentType)
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	// Parse multipart form data
	mr, err := r.MultipartReader()
	if err != nil {
		fmt.Println("Error creating multipart reader:", err)
		http.Error(w, "Failed to read multipart data", http.StatusBadRequest)
		return
	}

	var seedBytes []byte

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading part:", err)
			http.Error(w, "Failed to read multipart data", http.StatusInternalServerError)
			return
		}

		switch part.FormName() {
		case "seed":
			seed, err := io.ReadAll(part)
			if err != nil || len(seed) == 0 {
				fmt.Println("Error reading seed field:", err)
				http.Error(w, "Invalid seed field", http.StatusBadRequest)
				return
			}
			seedBytes = seed
		default:
			fmt.Println("Unknown form field:", part.FormName())
		}
	}

	if len(seedBytes) == 0 {
		fmt.Println("Seed not provided")
		http.Error(w, "Seed not provided", http.StatusBadRequest)
		return
	}

	// Seal and store
	sealedSeed, err := sealing.NewSealing(seedBytes, []byte(""), true)
	if err != nil {
		fmt.Println("Error sealing seed:", err)
		http.Error(w, "Failed to seal seed", http.StatusInternalServerError)
		return
	}

	if err := seedGeneration.WriteSealedSeed(sealedSeed); err != nil {
		fmt.Println("Error storing sealed seed:", err)
		http.Error(w, "Failed to store sealed seed", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write([]byte("Seed stored successfully"))
}

// -------------------------
// Redis request processing
// -------------------------
type EncryptedPayload struct {
	ClientID string ` + "`json:\"client_id\"`" + `
	Data     string ` + "`json:\"data\"`" + `
}

type ClientPayload struct {
	FunctionName    string ` + "`json:\"function_name\"`" + `
	Certificate     string ` + "`json:\"certificate\"`" + `
	EncryptedData   string ` + "`json:\"encrypted_data\"`" + `
	FileExtension   string ` + "`json:\"file_extension\"`" + `
	IPNSKey         string ` + "`json:\"ipns_key\"`" + `
	ClientPublicKey string ` + "`json:\"client_public_key\"`" + `
	Signature       string ` + "`json:\"signature\"`" + `
}

type QueuePayload struct {
	ClientID string        ` + "`json:\"client_id\"`" + `
	Data     ClientPayload ` + "`json:\"data\"`" + `
}

func readQueue() {
	for {
		res, err := redisClient.BRPop(redisClient.Context(), 0, requestQueue).Result()
		if err != nil {
			fmt.Printf("[Server] Error dequeuing request: %v\n", err)
			continue
		}

		var payload QueuePayload
		if err := json.Unmarshal([]byte(res[1]), &payload); err != nil {
			fmt.Printf("[Server] Error parsing payload: %v\n", err)
			continue
		}

		response := handleFunction(payload.Data.FunctionName, map[string]string{
			"function_name":     payload.Data.FunctionName,
			"certificate":       payload.Data.Certificate,
			"encrypted_data":    payload.Data.EncryptedData,
			"file_extension":    payload.Data.FileExtension,
			"ipns_key":          payload.Data.IPNSKey,
			"client_public_key": payload.Data.ClientPublicKey,
			"signature":         payload.Data.Signature,
		})

		responsePayload, _ := json.Marshal(map[string]string{
			"client_id": payload.ClientID,
			"response":  response,
		})
		if err := redisClient.LPush(redisClient.Context(), responseQueue, responsePayload).Err(); err != nil {
			fmt.Printf("[Server] Error enqueuing response: %v\n", err)
		} else {
			fmt.Printf("[Server] Enqueued response for client: %s\n", payload.ClientID)
			fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
		}
	}
}
`)

	packages := fmt.Sprintf(`package tee_server_receiver
`)

	// Imports: keep only what the generated file really needs now
	imports := fmt.Sprintf(`
import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	teeRequester "sparta/src/teeserver/requester"
	"sparta/src/utils/interfaceISGoMiddleware"
	"sparta/src/utils/sealing"
	seedGeneration "sparta/src/utils/seedGenerator"

	"github.com/edgelesssys/ego/enclave"
	"github.com/go-redis/redis/v8"
)
	`)

	return packages + imports + serverScaffold + handleFunction + functions
}