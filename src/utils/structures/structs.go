package structs

import "reflect"

var StructRegistry = map[string]reflect.Type{
	"Patient": reflect.TypeOf(Patient{}),
	"PatientLight": reflect.TypeOf(PatientLight{}),
	// Add other structs here if needed
}

type Patient struct {
	PhoneNumber string `json:"PhoneNumber"`
	Occupation string `json:"Occupation"`
	Diet string `json:"Diet"`
	EmergencyContactPhone string `json:"EmergencyContactPhone"`
	LastPhysicalExamDate string `json:"LastPhysicalExamDate"`
	BodyMassIndex float64 `json:"BodyMassIndex"`
	BloodType string `json:"BloodType"`
	TravelHistory []string `json:"TravelHistory"`
	Email string `json:"Email"`
	FamilyMedicalHistory []string `json:"FamilyMedicalHistory"`
	EmergencyContactName string `json:"EmergencyContactName"`
	DateOfBirth string `json:"DateOfBirth"`
	PreviousVaccinations []string `json:"PreviousVaccinations"`
	InformedConsentDocument bool `json:"InformedConsentDocument"`
	Height float64 `json:"Height"`
	Weight float64 `json:"Weight"`
	PrimaryCarePhysician string `json:"PrimaryCarePhysician"`
	ID float64 `json:"ID"`
	IdentificationNumber float64 `json:"IdentificationNumber"`
	Age float64 `json:"Age"`
	Gender string `json:"Gender"`
	Address string `json:"Address"`
	CurrentMedications []string `json:"CurrentMedications"`
	ConsentFormSigned bool `json:"ConsentFormSigned"`
	PrivacyAgreement bool `json:"PrivacyAgreement"`
	OrganDonorStatus bool `json:"OrganDonorStatus"`
	CholesterolLevel string `json:"CholesterolLevel"`
	PreExistingConditions []string `json:"PreExistingConditions"`
	Ethnicity string `json:"Ethnicity"`
	MaritalStatus string `json:"MaritalStatus"`
	AnnualIncome float64 `json:"AnnualIncome"`
	AlcoholConsumption string `json:"AlcoholConsumption"`
	ExerciseFrequency string `json:"ExerciseFrequency"`
	BloodPressure string `json:"BloodPressure"`
	FullName string `json:"FullName"`
	EducationLevel string `json:"EducationLevel"`
	SmokingStatus string `json:"SmokingStatus"`
	InsuranceProvider string `json:"InsuranceProvider"`
	MeanAge float64 `json:"MeanAge"`
	SumAge float64 `json:"SumAge"`
}

type PatientLight struct {
	Age float64 `json:"Age"`
	PreExistingConditions []string `json:"PreExistingConditions"`
	CurrentMedications []string `json:"CurrentMedications"`
	PreviousVaccinations []string `json:"PreviousVaccinations"`
	FamilyMedicalHistory []string `json:"FamilyMedicalHistory"`
	ConsentFormSigned bool `json:"ConsentFormSigned"`
	MeanAge float64 `json:"MeanAge"`
	SumAge float64 `json:"SumAge"`
}

