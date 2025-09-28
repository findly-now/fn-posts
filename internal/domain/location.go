package domain

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrInvalidLatitude  = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude = errors.New("longitude must be between -180 and 180")
)

type Location struct {
	Latitude  float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" binding:"required,min=-180,max=180"`
}

type Distance struct {
	Meters float64 `json:"meters"`
}

func NewLocation(latitude, longitude float64) (Location, error) {
	loc := Location{
		Latitude:  latitude,
		Longitude: longitude,
	}

	if err := loc.Validate(); err != nil {
		return Location{}, err
	}

	return loc, nil
}

func (l Location) Validate() error {
	if l.Latitude < -90 || l.Latitude > 90 {
		return ErrInvalidLatitude
	}

	if l.Longitude < -180 || l.Longitude > 180 {
		return ErrInvalidLongitude
	}

	return nil
}

func (l Location) DistanceTo(other Location) Distance {
	const earthRadiusKm = 6371.0

	lat1Rad := l.Latitude * math.Pi / 180
	lat2Rad := other.Latitude * math.Pi / 180
	deltaLatRad := (other.Latitude - l.Latitude) * math.Pi / 180
	deltaLngRad := (other.Longitude - l.Longitude) * math.Pi / 180

	a := math.Sin(deltaLatRad/2)*math.Sin(deltaLatRad/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLngRad/2)*math.Sin(deltaLngRad/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distanceKm := earthRadiusKm * c
	distanceMeters := distanceKm * 1000

	return Distance{Meters: distanceMeters}
}

func (l Location) IsWithinRadius(other Location, radiusMeters float64) bool {
	distance := l.DistanceTo(other)
	return distance.Meters <= radiusMeters
}

func (d Distance) ToKilometers() float64 {
	return d.Meters / 1000
}

func (d Distance) ToMiles() float64 {
	return d.Meters * 0.000621371
}

func (d Distance) String() string {
	if d.Meters >= 1000 {
		return fmt.Sprintf("%.2f km", d.ToKilometers())
	}
	return fmt.Sprintf("%.0f m", d.Meters)
}

func (d Distance) Equals(other Distance) bool {
	return d.Meters == other.Meters
}

func (d Distance) IsZero() bool {
	return d.Meters == 0
}
