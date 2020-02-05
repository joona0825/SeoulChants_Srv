package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    _ "github.com/go-sql-driver/mysql"
    "net/http"
    "os"
    "strings"

    "log"
)

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
    Date        string      `json:"date"`
    Result      *string     `json:"result"`
    Highlight   *string     `json:"highlight"`
    Competition string      `json:"competition"`
    Round       string      `json:"round"`
    Home        bool        `json:"home"`
    Lineup      *string     `json:"lineup"`
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

// Response 서버 백엔드 결과를 반환하기 위한 구조체
type Response struct {
    Result      bool        `json:"result"`
    Data        interface{} `json:"data"`
}




func init() {
    f, err := os.OpenFile("/home/joona0825/seoulchants_srv.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
       fmt.Println("Log file not found!")
    } else {
        log.SetOutput(f)
    }

    log.Println("instance is now running!")
}

func main() {
    http.HandleFunc("/seoulchants/register/", registerToken)
    http.HandleFunc("/seoulchants/list/", list)
    http.HandleFunc("/seoulchants/info/", info)
    http.HandleFunc("/seoulchants/matches/", matches)
    http.HandleFunc("/seoulchants/matches/next/", nextMatch)
    http.ListenAndServe(":9090", nil)
    log.Println("main() server started...")
}

func database() *sql.DB {
    db, err := sql.Open("mysql", "services:XYDBFpZDQ9TG1YDz@tcp(127.0.0.1:3306)/services")
    if err != nil {
        log.Println("Failed to open database!! " + err.Error())
        return nil
    }

    return db
}

// 토큰 등록
func registerToken(w http.ResponseWriter, request *http.Request) {
    token := request.PostFormValue("token")
    if len(token) > 0 {
        db := database()
        if db == nil {
            internalErrorHandler(w, "db is nil")
            return
        }
        defer db.Close()

        var len int
        // 저장된 토큰이 있는지 여부 판단
        db.QueryRow("select COUNT(*) from `seoul_chants_tokens` where `token` = ?", token).Scan(&len)

        var query string
        if len == 0 {
            // 기존에 등록된 토큰이 없음 -> insert 필요
            query = "insert into `seoul_chants_tokens` (`token`, `last_active`) values (?, now())"
        } else {
            // 기존에 등록된 토큰이 있음 -> last_active만 업데이트
            query = "update `seoul_chants_tokens` set `last_active` = now() where `token` = ?"
        }

        _, err := db.Exec(query, token)
        if err != nil {
            internalErrorHandler(w, "execError" + err.Error())
            return
        }

        success(w, nil)
        if len == 0 {
            log.Println("new device registered: " + token)
        } else {
            log.Println("device updated: " + token)
        }

    } else {
        internalErrorHandler(w, "token is empty")
        return
    }
}

// 응원가 목록
func list(w http.ResponseWriter, request *http.Request) {
    db := database()
    if db == nil {
        internalErrorHandler(w, "db is nil")
        return
    }
    defer db.Close()

    var query string

    // 어떤 리스트를 요청한건지
    path := strings.Replace(request.URL.Path, "/seoulchants/list/", "", 1)
    if path == "chants" {
        // 응원가
        query = "select * from `seoul_chants` where `ord` < 1000 order by `ord` asc"
    } else if path == "playercall" {
        // 선수 콜
        query = "select * from `seoul_chants` where `ord` >= 1000 order by `ord` asc"
    } else {
        notFoundHandler(w, request)
        return
    }

    rows, err := db.Query(query)
    if err != nil {
        internalErrorHandler(w, "list " + err.Error())
        return
    }
    defer rows.Close()


    var songs []Song

    for rows.Next() {
        var song Song
        err := rows.Scan(&song.ID, &song.ord, &song.Name, &song.Lyrics, &song.Etc, &song.Youtube, &song.Asset, &song.Hot, &song.New)
        if err == nil {
            songs = append(songs, song)
        } else {
            log.Println("list error: " + err.Error())
        }
    }

    success(w, songs)
}

// 단일 응원가 내용
func info(w http.ResponseWriter, request *http.Request) {
    db := database()
    if db == nil {
        internalErrorHandler(w, "db is nil")
        return
    }
    defer db.Close()

    path := strings.Replace(request.URL.Path, "/seoulchants/info/", "", 1)

    var song Song
    err := db.QueryRow("select * from `seoul_chants` where `id` = ?", path).Scan(&song.ID, &song.ord, &song.Name, &song.Lyrics, &song.Etc, &song.Youtube, &song.Asset, &song.Hot, &song.New)
    if err != nil {
        notFoundHandler(w, request)
        return
    }

    success(w, song)
}

// 경기 목록
func matches(w http.ResponseWriter, request *http.Request) {
    const season = 2020

    db := database()
    if db == nil {
        internalErrorHandler(w, "db is nil")
        return
    }
    defer db.Close()

    // 축악어 로드가 필요하면 리스트 불러오기
    var abbr = make(map[string]*string)
    abb, ok := request.URL.Query()["abb"]
    if ok && abb[0] == "1" {
        rows, err := db.Query("select `name`, `abb` from `seoul_chants_shortcut`")
        if err != nil {
            internalErrorHandler(w, "abb " + err.Error())
            return
        }
        defer rows.Close()

        for rows.Next() {
            var name string
            var abb *string

            err := rows.Scan(&name, &abb)
            if err == nil {
                abbr[name] = abb
            }
        }
    }

    rows, err := db.Query("select * from `seoul_chants_matches` where year(`date`) = ? order by `date` desc", season)
    if err != nil {
        internalErrorHandler(w, "matches " + err.Error())
        return
    }
    defer rows.Close()

    var matches []Match

    for rows.Next() {
        var match Match

        // location이 0이면 원정 경기장, 1이면 홈 경기장, 나머지면 alternative stadium
        var location int

        err := rows.Scan(&match.id, &match.Vs, &match.Date, &match.Result, &match.Highlight, &match.Competition, &match.Round, &location, &match.Lineup, &match.messageSent, &match.Preview)
        if err == nil {
            match.Home = (location == 1)
            match.Abb = abbr[match.Vs]
            matches = append(matches, match)
        } else {
            log.Println("matches error: " + err.Error())
        }
    }

    success(w, MatchesResponse{Season: season, Matches: matches})

}

// 다음 경기 정보
func nextMatch(w http.ResponseWriter, request *http.Request) {
    db := database()
    if db == nil {
        internalErrorHandler(w, "db is nil")
        return
    }
    defer db.Close()

    var match Match
    var location int    // location이 0이면 원정 경기장, 1이면 홈 경기장, 나머지면 alternative stadium

    err := db.QueryRow("select * from `seoul_chants_matches` where `date` > date_sub(now(), interval 2 hour) order by `date` asc limit 0,1").Scan(&match.id, &match.Vs, &match.Date, &match.Result, &match.Highlight, &match.Competition, &match.Round, &location, &match.Lineup, &match.messageSent, &match.Preview)

    path := strings.Replace(request.URL.Path, "/seoulchants/matches/next/", "", 1)
    if len(path) != 0 {
        // 특정한 id의 일정 구해오기
        err = db.QueryRow("select * from `seoul_chants_matches` where `id` = ? limit 0,1", path).Scan(&match.id, &match.Vs, &match.Date, &match.Result, &match.Highlight, &match.Competition, &match.Round, &location, &match.Lineup, &match.messageSent, &match.Preview)
    }

    if err != nil {
        // 다음 경기가 없음
        success(w, nil)
        return
    }

    match.Home = (location == 1)

    // 경기 장소
    var stadium Stadium
    if match.Home {
        // 홈인 경우 고정된 값 사용
        stadium.Name = "서울월드컵경기장"
        stadium.Latitude = 37.5682588
        stadium.Longitude = 126.8972774
    } else {
        // 원정인 경우 스타디움 가져오기
        // location이 0이면 상대팀의 기본 홈 구장, 다른 값이면 alternative 구장
        err := db.QueryRow("select * from `seoul_chants_stadiums` where `team` = ? and `alternative` = ? limit 0,1", match.Vs, (location != 0)).Scan(&stadium.id, &stadium.Name, &stadium.Latitude, &stadium.Longitude, &stadium.team, &stadium.alternative)
        if err != nil {
            // 경기장 정보를 받아오지 못함 -> 그냥 원정이라고만 표기
            stadium.Name = "원정"
        }
    }
    match.Stadium = &stadium

    // 이전 경기 목록
    rows, _ := db.Query("select `date`, `result`, `highlight`, `competition`, `round` from `seoul_chants_matches` where `vs` = ? and `date` < ? order by `date` desc limit 0,5", match.Vs, match.Date)
    defer rows.Close()

    var previousMatches []Match

    for rows.Next() {
        var match Match
        err := rows.Scan(&match.Date, &match.Result, &match.Highlight, &match.Competition, &match.Round)
        if err == nil {
            previousMatches = append(previousMatches, match)
        } else {
            log.Println("nextMatch previous match error: " + err.Error())
        }
    }

    match.Previous = previousMatches

    success(w, match)
}


// 200 성공
func success(w http.ResponseWriter, data interface{}) {
    w.WriteHeader(http.StatusOK)

    response, _ := json.Marshal(Response{Result: true, Data: data})
    fmt.Fprint(w, string(response))
}

// 404 에러 핸들러
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    fmt.Fprint(w, "404 page not found")

    log.Println("404 page not found: access tried to " + r.RequestURI)
}

// 500 에러 핸들러
func internalErrorHandler(w http.ResponseWriter, message string) {
    w.WriteHeader(http.StatusInternalServerError)

    response, _ := json.Marshal(Response{Result: false, Data: "서버 내부 오류가 발생했습니다."})
    fmt.Fprint(w, string(response))

    log.Println("internalError: " + message)
}
