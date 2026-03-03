package decisionfunctions

import (
    "math"
)

type PatientPriorityDecision struct{}

func (d PatientPriorityDecision) PatientPriority(inputs map[string]interface{}) (string, float64, string, float64, bool) {
    if inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 60 && (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol" || inputs["CurrentMedications"].(string) == "Lisinopril") {
        return "High", 5, "Low", 1, true
    }
    if (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol" || inputs["CurrentMedications"].(string) == "Lisinopril") && (inputs["PreviousVaccinations"].(string) == "COVID-19" || inputs["PreviousVaccinations"].(string) == "Influenza") && (inputs["FamilyMedicalHistory"].(string) == "Diabetes" || inputs["FamilyMedicalHistory"].(string) == "Heart Disease") && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 60 {
        return "High", 5, "Low", 2, true
    }
    if inputs["Age"].(float64) >= 60 && (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && (inputs["PreviousVaccinations"].(string) == "COVID-19" || inputs["PreviousVaccinations"].(string) == "Influenza") && (inputs["FamilyMedicalHistory"].(string) == "Diabetes" || inputs["FamilyMedicalHistory"].(string) == "Heart Disease") && inputs["ConsentFormSigned"].(bool) == true {
        return "High", 5, "Low", 3, true
    }
    if (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol" || inputs["CurrentMedications"].(string) == "Lisinopril") && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 60 {
        return "High", 5.2, "Low", 4, true
    }
    if inputs["Age"].(float64) >= 60 && (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && inputs["ConsentFormSigned"].(bool) == true {
        return "High", 51, "Low", 5, false
    }
    if inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 60 {
        return "Low", 5, "Low", 6, false
    }
    if (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol" || inputs["CurrentMedications"].(string) == "Lisinopril") && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 {
        return "High", 5, "Low", 7, false
    }
    if inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol" || inputs["CurrentMedications"].(string) == "Lisinopril") && (inputs["PreviousVaccinations"].(string) == "COVID-19" || inputs["PreviousVaccinations"].(string) == "Influenza") && (inputs["FamilyMedicalHistory"].(string) == "Diabetes" || inputs["FamilyMedicalHistory"].(string) == "Heart Disease") {
        return "Medium", 5, "Low", 8, false
    }
    if inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && (inputs["PreviousVaccinations"].(string) == "COVID-19" || inputs["PreviousVaccinations"].(string) == "Influenza") && (inputs["FamilyMedicalHistory"].(string) == "Diabetes" || inputs["FamilyMedicalHistory"].(string) == "Heart Disease") && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium", 5, "Low", 9, false
    }
    if inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol" || inputs["CurrentMedications"].(string) == "Lisinopril") {
        return "Medium", 5, "Low", 8, false
    }
    if inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 && (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium", 5, "Low", 8, false
    }
    if inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) >= 18 && inputs["Age"].(float64) < 60 {
        return "Low", 5, "Low", 8, false
    }
    if inputs["Age"].(float64) < 18 && (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol" || inputs["CurrentMedications"].(string) == "Lisinopril") && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium", 5, "Low", 8, false
    }
    if (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol" || inputs["CurrentMedications"].(string) == "Lisinopril") && (inputs["PreviousVaccinations"].(string) == "COVID-19" || inputs["PreviousVaccinations"].(string) == "Influenza") && (inputs["FamilyMedicalHistory"].(string) == "Diabetes" || inputs["FamilyMedicalHistory"].(string) == "Heart Disease") && inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) < 18 {
        return "Medium", 5, "Low", 8, false
    }
    if inputs["Age"].(float64) < 18 && (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && (inputs["PreviousVaccinations"].(string) == "COVID-19" || inputs["PreviousVaccinations"].(string) == "Influenza") && (inputs["FamilyMedicalHistory"].(string) == "Diabetes" || inputs["FamilyMedicalHistory"].(string) == "Heart Disease") && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium", 5, "Low", 8, false
    }
    if inputs["Age"].(float64) < 18 && (inputs["CurrentMedications"].(string) == "Metformin" || inputs["CurrentMedications"].(string) == "Albuterol" || inputs["CurrentMedications"].(string) == "Lisinopril") && inputs["ConsentFormSigned"].(bool) == true {
        return "Medium", 5, "Low", 8, true
    }
    if inputs["Age"].(float64) < 18 && (inputs["PreExistingConditions"].(string) == "Asthma" || inputs["PreExistingConditions"].(string) == "Diabetes") && inputs["ConsentFormSigned"].(bool) == true {
        return "Low", 5, "Low", 8, true
    }
    if inputs["ConsentFormSigned"].(bool) == true && inputs["Age"].(float64) < 18 {
        return "Low", 5, "Low", 8, true
    }
    if inputs["ConsentFormSigned"].(bool) == false {
        return "Ineligible", 9.4, "Low", 8, true
    }
    return "", math.NaN(), "", math.NaN(), false
}
