package model

type User struct {
	ID    string
	Email string
}

type Profile struct {
	Handle       string
	BirthYear    *int
	BirthMonth   *int16
	CountryCode  *string
	Timezone     string
	Discoverable bool
}

type Language struct {
	LanguageCode string
	Level        int16
	IsNative     bool
	IsTarget     bool
	Description  *string
}

type AvailabilitySlot struct {
	Weekday        int16
	StartLocalTime string
	EndLocalTime   string
	Timezone       string
}
