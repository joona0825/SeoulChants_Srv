package main

import (
    "database/sql"
    "fmt"
    _ "github.com/go-sql-driver/mysql"
    "net/http"
    "os"
    "strings"
    "time"

    "log"
)

const YEAR = 2021

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
    http.HandleFunc("/seoulchants/matches/", matches)
    http.HandleFunc("/seoulchants/matches/next/", nextMatch)
    http.HandleFunc("/seoulchants/player-history/", playerHistory)
    http.ListenAndServe(":9090", nil)
    log.Println("main() server started...")
}

func database() *sql.DB {
    db, err := sql.Open("mysql", DB_USERNAME + ":" + DB_PASSWORD + "@tcp(127.0.0.1:3306)/" + DB_DATABASE)
    if err != nil {
        log.Println("Failed to open database!! " + err.Error())
        return nil
    }

    return db
}

// 토큰 등록
func registerToken(w http.ResponseWriter, request *http.Request) {
    token := request.PostFormValue("token")
    tokenType := request.PostFormValue("type")

    // 기본 firebase
    if len(tokenType) == 0 {
        tokenType = "firebase"
    }

    if len(token) > 0 {
        db := database()
        if db == nil {
            internalErrorHandler(w, "db is nil")
            return
        }
        defer db.Close()

        var len int

        // 저장된 토큰이 있는지 여부 판단
        db.QueryRow("select COUNT(*) from `seoul_chants_tokens` where `token` = ? and `type` = ?", token, tokenType).Scan(&len)

        if len == 0 {
            // 기존에 등록된 토큰이 없음 -> insert 필요
            _, err := db.Exec("insert into `seoul_chants_tokens` (`type`, `token`, `last_active`) values (?, ?, now())", tokenType, token)
            if err != nil {
                internalErrorHandler(w, "execError " + err.Error())
                return
            }
        } else {
            // 기존에 등록된 토큰이 있음 -> last_active만 업데이트
            _, err := db.Exec("update `seoul_chants_tokens` set `last_active` = now() where `token` = ? and `type` = ?", token, tokenType)
            if err != nil {
                internalErrorHandler(w, "execError" + err.Error())
                return
            }
        }

        success(w, nil, request.RequestURI)
        if len == 0 {
            log.Printf("new device registetred: %s (%s)\n", token, tokenType)
        } else {
            log.Printf("        device updated: %s (%s)\n", token, tokenType)
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
        query = "select `id`, `ord`, `name`, `lyrics`, `etc`, `youtube`, `asset`, `hot`, `new` from `seoul_chants` where `ord` between 0 and 999 order by `ord` asc"
    } else if path == "playercall" {
        // 선수 콜
        query = "select `id`, `ord`, `name`, `lyrics`, `etc`, `youtube`, `asset`, `hot`, `new` from `seoul_chants` where `ord` >= 1000 order by `ord` asc"
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

    songs := make([]Song, 0)

    for rows.Next() {
        var song Song
        err := rows.Scan(&song.ID, &song.ord, &song.Name, &song.Lyrics, &song.Etc, &song.Youtube, &song.Asset, &song.Hot, &song.New)
        if err == nil {
            songs = append(songs, song)
        } else {
            log.Println("list error: " + err.Error())
        }
    }

    success(w, songs, request.RequestURI)
}

// 경기 목록
func matches(w http.ResponseWriter, request *http.Request) {
    db := database()
    if db == nil {
        internalErrorHandler(w, "db is nil")
        return
    }
    defer db.Close()

    // 축악어 로드가 필요하면 리스트 불러오기
    var abbr = make(map[string]*string)
    _abb, hasQuery := request.URL.Query()["abb"]
    if hasQuery && _abb[0] == "1" {
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

    rows, err := db.Query("select `id`, `vs`, `date`, `dateTBA`, `timeTBA`, `result`, `highlight`, `competition`, `round`, `home`  from `seoul_chants_matches` where year(`date`) = ? order by `date` desc", YEAR)
    if err != nil {
        internalErrorHandler(w, "matches " + err.Error())
        return
    }
    defer rows.Close()

    matches := make([]Match, 0)

    for rows.Next() {
        var match Match

        // location이 0이면 원정 경기장, 1이면 홈 경기장, 나머지면 alternative stadium
        var location int
        var datetimeString string
        var dateTBA, timeTBA bool
        err := rows.Scan(&match.id, &match.Vs, &datetimeString, &dateTBA, &timeTBA, &match.Result, &match.Highlight, &match.Competition, &match.Round, &location)
        if err == nil {
            match.Home = (location == 1)
            match.Abb = abbr[match.Vs]
            match.Lineup = nil      // 데이터 아끼기 위해 라인업은 생략하기..

            datetimeTime, err := time.Parse("2006-01-02 15:04:05", datetimeString)
            if err == nil {
                dateString := datetimeTime.Format("2006-01-02")
                timeString := datetimeTime.Format("15:04:05")

                if dateTBA {
                    match.Date = nil
                    match.Time = nil
                } else if timeTBA {
                    match.Date = &dateString
                    match.Time = nil
                } else {
                    match.Date = &dateString
                    match.Time = &timeString
                }
            } else {
                log.Println(err.Error())
                match.Date = nil
                match.Time = nil
            }

            // 구버전 호환을 위하여 date는 일단 overwrite 하도록 함
            match.Date = &datetimeString

            matches = append(matches, match)
        } else {
            log.Println("matches error: " + err.Error())
        }
    }

    success(w, MatchesResponse{Season: YEAR, Matches: matches}, request.RequestURI)

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

    path := strings.Replace(request.URL.Path, "/seoulchants/matches/next/", "", 1)

    var err error
    var datetimeString string
    var dateTBA, timeTBA bool

    log.Printf("path: %s len %d", path, len(path))
    if len(path) != 0 {
        // 특정한 id의 일정 구해오기
        err = db.QueryRow("select `id`, `vs`, `date`, `dateTBA`, `timeTBA`, `result`, `highlight`, `competition`, `round`, `home`, `lineup`, `lineup_sub`, `message_sent`, `preview_available` from `seoul_chants_matches` where `id` = ? limit 0,1", path).
            Scan(&match.id, &match.Vs, &datetimeString, &dateTBA, &timeTBA, &match.Result, &match.Highlight, &match.Competition, &match.Round, &location, &match.Lineup, &match.LineupSub, &match.messageSent, &match.Preview)
    } else {
        err = db.QueryRow("select `id`, `vs`, `date`, `dateTBA`, `timeTBA`, `result`, `highlight`, `competition`, `round`, `home`, `lineup`, `lineup_sub`, `message_sent`, `preview_available` from `seoul_chants_matches` where `date` > date_sub(now(), interval 2 hour) order by `date` asc limit 0,1").
            Scan(&match.id, &match.Vs, &datetimeString, &dateTBA, &timeTBA, &match.Result, &match.Highlight, &match.Competition, &match.Round, &location, &match.Lineup, &match.LineupSub, &match.messageSent, &match.Preview)
    }

    if err != nil {
        // 다음 경기가 없음
        log.Println(err.Error())
        success(w, nil, request.RequestURI)
        return
    }

    match.Home = (location == 1)

    datetimeTime, err := time.Parse("2006-01-02 15:04:05", datetimeString)
    if err == nil {
        dateString := datetimeTime.Format("2006-01-02")
        timeString := datetimeTime.Format("15:04:05")

        if dateTBA {
            match.Date = nil
            match.Time = nil
        } else if timeTBA {
            match.Date = &dateString
            match.Time = nil
        } else {
            match.Date = &dateString
            match.Time = &timeString
        }
    } else {
        log.Println(err.Error())
        match.Date = nil
        match.Time = nil
    }

    // 구버전 호환을 위하여 date는 일단 overwrite 하도록 함
    match.Date = &datetimeString

    // 경기 장소
    var stadium Stadium
    // covid-19로 인해 중립경기로 열리는 경기들의 경기장 정보
    if match.id == 229 {
        // vs 베이징 home
        stadium.Name = "Education City Stadium"
        stadium.Latitude = 25.3107835
        stadium.Longitude = 51.4222389
    } else if match.id == 230 {
        // vs 치앙라이 home
        stadium.Name = "Jassim bin Hamad Stadium"
        stadium.Latitude = 25.2674291
        stadium.Longitude = 51.4842975
    } else if match.id == 231 {
        // vs 치앙라이 away
        stadium.Name = "Jassim bin Hamad Stadium"
        stadium.Latitude = 25.2674291
        stadium.Longitude = 51.4842975
    } else if match.id == 232 {
        // vs 베이징 away
        stadium.Name = "Jassim bin Hamad Stadium"
        stadium.Latitude = 25.2674291
        stadium.Longitude = 51.4842975
    } else if match.id == 233 {
        // vs 멜버른 away
        stadium.Name = "Education City Stadium"
        stadium.Latitude = 25.3107835
        stadium.Longitude = 51.4222389
    } else {
        if match.Home {
            // 홈인 경우 고정된 값 사용
            stadium.Name = "서울월드컵경기장"
            stadium.Latitude = 37.5682588
            stadium.Longitude = 126.8972774
        } else {
            // 원정인 경우 스타디움 가져오기
            // location이 0이면 상대팀의 기본 홈 구장, 다른 값이면 alternative 구장
            err := db.QueryRow("select `id`, `name`, `latitude`, `longitude` from `seoul_chants_stadiums` where `team` = ? and `alternative` = ? limit 0,1", match.Vs, (location != 0)).
                Scan(&stadium.id, &stadium.Name, &stadium.Latitude, &stadium.Longitude)
            if err != nil {
                // 경기장 정보를 받아오지 못함 -> 그냥 원정이라고만 표기
                stadium.Name = "원정"
            }
        }
    }
    match.Stadium = &stadium

    // 이전 경기 목록
    rows, _ := db.Query("select `date`, `result`, `highlight`, `competition`, `round` from `seoul_chants_matches` where `vs` = ? and YEAR(`date`) > 1983 and `date` < ? order by `date` desc limit 0,5", match.Vs, match.Date)
    defer rows.Close()

    previousMatches := make([]Match, 0)

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

    success(w, match, request.RequestURI)
}

// 선수 출장 기록
func playerHistory(w http.ResponseWriter, request *http.Request) {
    db := database()
    if db == nil {
        internalErrorHandler(w, "db is nil")
        return
    }
    defer db.Close()

    _player, hasQuery := request.URL.Query()["name"]
    if !hasQuery || len(_player[0]) == 0 {
        log.Println("player is empty")
        internalErrorHandler(w, "player is empty")
        return
    }

    player := _player[0]

    starting := make([]PlayerHistoryMatch, 0)
    sub := make([]PlayerHistoryMatch, 0)

    // 선발 조회
    startingAppearanceRows, err := db.Query("select vs, competition, round from `seoul_chants_matches` where `lineup` like ? and YEAR(`date`) = ?", "%"+player+"%", YEAR)
    if err == nil {
        for startingAppearanceRows.Next() {
            var match PlayerHistoryMatch
            err := startingAppearanceRows.Scan(&match.Vs, &match.Competition, &match.Round)
            if err == nil {
                starting = append(starting, match)
            } else {
                log.Println("startingAppearance error: " + err.Error())
            }
        }
        startingAppearanceRows.Close()
    } else {
        log.Println("startingAppearance error: " + err.Error())
    }

    // 교체 조회
    subAppearanceRows, err := db.Query("select vs, competition, round from `seoul_chants_matches` where `lineup_sub` like ? and YEAR(`date`) = ?", "%"+player+"%", YEAR)
    if err == nil {
        for subAppearanceRows.Next() {
            var match PlayerHistoryMatch
            err := subAppearanceRows.Scan(&match.Vs, &match.Competition, &match.Round)
            if err == nil {
                sub = append(sub, match)
            } else {
                log.Println("subAppearance error: " + err.Error())
            }
        }
        subAppearanceRows.Close()
    } else {
        log.Println("subAppearance error: " + err.Error())
    }

    var response PlayerHistoryResponse
    response.Starting = starting
    response.Sub = sub

    success(w, response, request.RequestURI)
}
