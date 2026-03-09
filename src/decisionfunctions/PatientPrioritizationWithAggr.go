package decisionfunctions


type PatientPrioritizationWithAggrDecision struct{}

func (d PatientPrioritizationWithAggrDecision) PatientPrioritizationWithAggr(inputs map[string]interface{}) (string) {
    if (inputs["PreviousVaccinations"].(string) == "COVID-19" || inputs["PreviousVaccinations"].(string) == "Influenza") && (inputs["FamilyMedicalHistory"].(string) == "Diabetes" || inputs["FamilyMedicalHistory"].(string) == "Heart Disease") && inputs["ConsentFormSigned"].(bool) == true && inputs["MeanAge"].(float64) >= 40 && inputs["SumAge"].(float64) > 250 && inputs["Age"].(float64) >= 60 && (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol") {
        return "High"
    }
    if inputs["ConsentFormSigned"].(bool) == true && inputs["MeanAge"].(float64) < 40 && inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && inputs["PreExistingConditions"].(string) == "Asthma" && inputs["CurrentMedications"].(string) == "Metformin" && inputs["PreviousVaccinations"].(string) == "Influenza" {
        return "Medium"
    }
    if inputs["CurrentMedications"].(string) == "Lisinopril" && inputs["PreviousVaccinations"].(string) == "COVID-19" && inputs["ConsentFormSigned"].(bool) == true && inputs["MeanAge"].(float64) > 18 && inputs["MeanAge"].(float64) < 60 && inputs["Age"].(float64) < 18 {
        return "Low"
    }
    if inputs["ConsentFormSigned"].(bool) == false {
        return "Ineligible"
    }
    return ""
}
