package main

// Song 응원가 단일 항목
type Song struct {
	ID          int         `json:"id"`
	ord         int
	Name        string      `json:"name"`
	Lyrics      string      `json:"lyrics"`
	Etc         string      `json:"etc"`
	Youtube     *string     `json:"youtube"`
	Asset       *string     `json:"asset"`
	Hot         bool        `json:"hot"`
	New         bool        `json:"new"`
}

// Stadium 경기 장소 정보
type Stadium struct {
	id          int
	Name        string      `json:"name"`
	Latitude    float32     `json:"latitude"`
	Longitude   float32     `json:"longitude"`
	team        string
	alternative bool
}

// Match 경기 정보
type Match struct {
	id          int
	Vs          string      `json:"vs"`
	Abb         *string     `json:"abb"`
	Date        *string     `json:"date"`
	Time        *string     `json:"time"`
	Result      *string     `json:"result"`
	Highlight   *string     `json:"highlight"`
	Competition string      `json:"competition"`
	Round       string      `json:"round"`
	Home        bool        `json:"home"`
	Lineup      *string     `json:"lineup"`
	LineupSub   *string     `json:"lineup_sub"`
	messageSent bool
	Preview     bool        `json:"preview_available"`

	Previous    []Match     `json:"previous"`
	Stadium     *Stadium    `json:"stadium"`
}

// MatchesResponse 경기 목록 결과를 반환하기 위한 구조체
type MatchesResponse struct {
	Season      int         `json:"season"`
	Matches     []Match     `json:"matches"`
}

// PlayerHistoryMatch 선수 출장기록 내려줄때 어떤 경기인지 정보를 담는 구조체
type PlayerHistoryMatch struct {
	Vs          string      `json:"vs"`
	Competition string      `json:"competition"`
	Round       string      `json:"round"`
}

// PlayerHistoryResponse 선수 출장기록 조회 결과 반환을 위한 구조체
type PlayerHistoryResponse struct {
	Starting    []PlayerHistoryMatch    `json:"starting"`
	Sub         []PlayerHistoryMatch    `json:"sub"`
}

// Response 서버 백엔드 결과를 반환하기 위한 구조체
type Response struct {
	Result      bool        `json:"result"`
	Data        interface{} `json:"data"`
	MinVersion  string      `json:"minVersion"`
}