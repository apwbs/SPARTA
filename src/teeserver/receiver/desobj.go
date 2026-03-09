package tee_server_receiver

import (
	"reflect"
	"sparta/src/utils/interfaceISGoMiddleware"
)

func PatientPrioritizationWithAggrHandler(payload map[string]string) string {
	certificate, functionName, ipnsKey, _ := interfaceISGoMiddleware.ParseDecisionRequestFromQueue(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`{accessPolicy: (Role = "MedicalHub" and Country = "Italy")}`, attributes)
		if callable {
			Additionals := make(map[string]interface{})
			structSliceInterface, _, _ := interfaceISGoMiddleware.RetrieveStructSliceLinkedLog("Patient", ipnsKey)
			structSlice, ok := structSliceInterface.(reflect.Value)
			if !ok {
				structSlice = reflect.ValueOf(structSliceInterface)
				if structSlice.Kind() != reflect.Slice {
					return "Error: Retrieved data is not a slice"
				}
			}
			aggrResult_meanAge, _, _ := interfaceISGoMiddleware.NewPerformAggregation("filteredPatients: Patient[ConsentFormSigned = true and contains(FamilyMedicalHistory, 'Heart Disease') and (Age <= 38 or (Age > 40 and contains(PreExistingConditions, 'Diabetes'))) and contains(PreviousVaccinations, 'COVID-19')], meanAge: mean(filteredPatients.Age)", structSlice)
			Additionals["meanAge"] = aggrResult_meanAge
			aggrResult_sumAge, _, _ := interfaceISGoMiddleware.NewPerformAggregation("filteredPatients: Patient[ConsentFormSigned = true], sumAge: sum(filteredPatients.Age)", structSlice)
			Additionals["sumAge"] = aggrResult_sumAge
			structSliceInterfaceForDecision, _, _ := interfaceISGoMiddleware.RetrieveStructSliceLinkedLog("PatientLight", ipnsKey+"Light")
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

func PatientPrioritizationMultipleOutputsHandler(payload map[string]string) string {
	certificate, functionName, ipnsKey, _ := interfaceISGoMiddleware.ParseDecisionRequestFromQueue(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`{accessPolicy: (Role = "MedicalHub" and Country = "Italy")}`, attributes)
		if callable {
			interfaceISGoMiddleware.Decision(functionName, "PatientLight", ipnsKey+"Light")
			return "Decision performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}

func PatientPrioritizationREHandler(payload map[string]string) string {
	certificate, functionName, ipnsKey, _ := interfaceISGoMiddleware.ParseDecisionRequestFromQueue(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`{accessPolicy: (Role = "MedicalHub" and Country = "Italy")}`, attributes)
		if callable {
			interfaceISGoMiddleware.Decision(functionName, "PatientLight", ipnsKey+"Light")
			return "Decision performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}

func WritePatientDataHandler(payload map[string]string) string {
	certificate, _, fileBytes, _, ipnsKey, _ := interfaceISGoMiddleware.ParseSetRequestFromQueueBytes(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`{accessPolicy: (Country = "Italy" and (Role = "MedicalHub" or Role = "VaccinationCenter"))}`, attributes)
		if callable {
			interfaceISGoMiddleware.EncryptAndUploadLinkedBytes(fileBytes, "Patient", ipnsKey)
			interfaceISGoMiddleware.EncryptAndUploadLinkedBytes(fileBytes, "PatientLight", ipnsKey+"Light")
			return "Encryption of document performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}
