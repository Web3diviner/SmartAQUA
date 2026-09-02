package computer_vision

// ImageFrame represents a grayscale image frame
type ImageFrame struct {
	Width  int       `json:"width"`
	Height int       `json:"height"`
	Data   [][]uint8 `json:"data"` // Grayscale pixel data (0-255)
}

// HSVColor represents a color in HSV color space
type HSVColor struct {
	H float64 `json:"h"` // Hue (0-360)
	S float64 `json:"s"` // Saturation (0-1)
	V float64 `json:"v"` // Value (0-1)
}

// Point represents a 2D point
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Vector2D represents a 2D vector
type Vector2D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// OpticalFlowVector represents motion vector between frames
type OpticalFlowVector struct {
	Point      Point    `json:"point"`      // Point location
	Velocity   Vector2D `json:"velocity"`   // Motion vector
	Magnitude  float64  `json:"magnitude"`  // Vector magnitude
	Confidence float64  `json:"confidence"` // Tracking confidence
}
