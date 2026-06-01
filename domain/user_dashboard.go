package domain

type UserDashboard struct {
	TotalClients           int64
	BlockedClients         int64
	NewClientsToday        int64
	NewClientsYesterday    int64
	NewClientsDeltaPercent float64
	ActiveClientsToday     int64
	UniqueVisitorsToday    int64
	Days                   []UserDashboardDay
}

type UserDashboardDay struct {
	Day            string
	NewClients     int64
	ActiveClients  int64
	UniqueVisitors int64
}
