package models

type DashboardRoom struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Omzet             float64 `json:"omzet"`
	PercentageOfUsage float64 `json:"percentageOfUsage"`
}

type DashboardResponse struct {
	Message string `json:"message"`
	Data    struct {
		TotalRoom        int             `json:"totalRoom"`
		TotalVisitor     int             `json:"totalVisitor"`
		TotalReservation int             `json:"totalReservation"`
		TotalOmzet       float64         `json:"totalOmzet"`
		Rooms            []DashboardRoom `json:"rooms"`
	} `json:"data"`
}
