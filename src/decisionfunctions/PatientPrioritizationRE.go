package decisionfunctions


type PatientPrioritizationREDecision struct{}

func (d PatientPrioritizationREDecision) PatientPrioritizationRE(inputs map[string]interface{}) (string) {
    if inputs["PreExistingConditions"].(string) == "Asthma" && inputs["CurrentMedications"].(string) == "Albuterol" && inputs["FamilyMedicalHistory"].(string) == "Parkinson's Disease" && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 60 {
        return "High1"
    }
    if inputs["PreviousVaccinations"].(string) == "Influenza" && inputs["FamilyMedicalHistory"].(string) == "Breast Cancer" && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 60 && inputs["PreExistingConditions"].(string) == "Hypertension" && inputs["CurrentMedications"].(string) == "Metformin" {
        return "High2"
    }
    if inputs["FamilyMedicalHistory"].(string) == "Diabetes" && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 60 && inputs["PreExistingConditions"].(string) == "Diabetes" && inputs["CurrentMedications"].(string) == "Prednisone" && inputs["PreviousVaccinations"].(string) == "Influenza" {
        return "High3"
    }
    if inputs["FamilyMedicalHistory"].(string) == "Cancer" && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 60 && inputs["PreExistingConditions"].(string) == "Hypertension" && inputs["CurrentMedications"].(string) == "Metformin" && inputs["PreviousVaccinations"].(string) == "Tetanus" {
        return "High4"
    }
    if inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 60 && inputs["CurrentMedications"].(string) == "Loratadine" && inputs["PreviousVaccinations"].(string) == "Lisinopril" && inputs["FamilyMedicalHistory"].(string) == "Chronic Lung Disease" {
        return "High5"
    }
    if inputs["Age"].(float64) >= 60 && inputs["ConsentFormSigned"].(bool) == true {
        return "Low1"
    }
    if inputs["PreviousVaccinations"].(string) == "Tetanus" && inputs["FamilyMedicalHistory"].(string) == "Mental Illness" && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && inputs["PreExistingConditions"].(string) == "Osteoarthritis" {
        return "High6"
    }
    if inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && inputs["PreExistingConditions"].(string) == "Obesity" && inputs["CurrentMedications"].(string) == "Ibuprofen" && inputs["PreviousVaccinations"].(string) == "Influenza" && inputs["FamilyMedicalHistory"].(string) == "Multiple Sclerosis" && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium1"
    }
    if inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && inputs["PreExistingConditions"].(string) == "Psoriasis" && inputs["CurrentMedications"].(string) == "Omeprazole" && inputs["PreviousVaccinations"].(string) == "Hib" && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium2"
    }
    if inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && inputs["PreExistingConditions"].(string) == "Depression" && inputs["CurrentMedications"].(string) == "Warfarin" && inputs["FamilyMedicalHistory"].(string) == "Breast Cancer" && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium3"
    }
    if inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && inputs["PreExistingConditions"].(string) == "Obesity" && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium4"
    }
    if inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && inputs["ConsentFormSigned"].(bool) == true {
        return "Low2"
    }
    if inputs["Age"].(float64) < 18 && inputs["PreExistingConditions"].(string) == "Hypertension" && inputs["CurrentMedications"].(string) == "Albuterol" && inputs["PreviousVaccinations"].(string) == "Rabies" && inputs["FamilyMedicalHistory"].(string) == "Multiple Sclerosis" && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium5"
    }
    if inputs["FamilyMedicalHistory"].(string) == "Autoimmune Diseases" && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) < 18 && inputs["CurrentMedications"].(string) == "Warfarin" {
        return "Medium6"
    }
    if inputs["FamilyMedicalHistory"].(string) == "Lung Cancer" && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) < 18 && inputs["PreviousVaccinations"].(string) == "Polio" {
        return "Medium7"
    }
    if inputs["FamilyMedicalHistory"].(string) == "Blood Disorders" && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) < 18 && inputs["PreviousVaccinations"].(string) == "Hepatitis A" {
        return "Medium8"
    }
    if inputs["Age"].(float64) < 18 && inputs["PreExistingConditions"].(string) == "Epilepsy" && inputs["ConsentFormSigned"].(bool) == true {
        return "Low3"
    }
    if inputs["Age"].(float64) < 18 && inputs["ConsentFormSigned"].(bool) == true {
        return "Low4"
    }
    if inputs["ConsentFormSigned"].(bool) == false {
        return "Ineligible"
    }
    return ""
}
