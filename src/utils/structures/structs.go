package structs

import "reflect"

var StructRegistry = map[string]reflect.Type{
	"Patient": reflect.TypeOf(Patient{}),
	"PatientLight": reflect.TypeOf(PatientLight{}),
	// Add other structs here if needed
}

type Patient struct {
	DateOfBirth string `json:"DateOfBirth"`
	Address string `json:"Address"`
	PreExistingConditions []string `json:"PreExistingConditions"`
	EducationLevel string `json:"EducationLevel"`
	AlcoholConsumption string `json:"AlcoholConsumption"`
	LastPhysicalExamDate string `json:"LastPhysicalExamDate"`
	CurrentMedications []string `json:"CurrentMedications"`
	MaritalStatus string `json:"MaritalStatus"`
	SmokingStatus string `json:"SmokingStatus"`
	EmergencyContactName string `json:"EmergencyContactName"`
	BodyMassIndex float64 `json:"BodyMassIndex"`
	FullName string `json:"FullName"`
	Ethnicity string `json:"Ethnicity"`
	Diet string `json:"Diet"`
	TravelHistory []string `json:"TravelHistory"`
	Age float64 `json:"Age"`
	Occupation string `json:"Occupation"`
	ExerciseFrequency string `json:"ExerciseFrequency"`
	PhoneNumber string `json:"PhoneNumber"`
	InformedConsentDocument bool `json:"InformedConsentDocument"`
	Weight float64 `json:"Weight"`
	ID float64 `json:"ID"`
	EmergencyContactPhone string `json:"EmergencyContactPhone"`
	PrimaryCarePhysician string `json:"PrimaryCarePhysician"`
	CholesterolLevel string `json:"CholesterolLevel"`
	Email string `json:"Email"`
	FamilyMedicalHistory []string `json:"FamilyMedicalHistory"`
	ConsentFormSigned bool `json:"ConsentFormSigned"`
	Height float64 `json:"Height"`
	InsuranceProvider string `json:"InsuranceProvider"`
	OrganDonorStatus bool `json:"OrganDonorStatus"`
	BloodPressure string `json:"BloodPressure"`
	Gender string `json:"Gender"`
	IdentificationNumber float64 `json:"IdentificationNumber"`
	PreviousVaccinations []string `json:"PreviousVaccinations"`
	PrivacyAgreement bool `json:"PrivacyAgreement"`
	BloodType string `json:"BloodType"`
	AnnualIncome float64 `json:"AnnualIncome"`
}

type PatientLight struct {
	Age float64 `json:"Age"`
	PreExistingConditions []string `json:"PreExistingConditions"`
	CurrentMedications []string `json:"CurrentMedications"`
	PreviousVaccinations []string `json:"PreviousVaccinations"`
	FamilyMedicalHistory []string `json:"FamilyMedicalHistory"`
	ConsentFormSigned bool `json:"ConsentFormSigned"`
}

