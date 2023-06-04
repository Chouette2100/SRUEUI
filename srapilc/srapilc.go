package srapilc

import (
	//	"bytes"
	"fmt"
	"log"

	//	"math"
	//	"sort"
	//	"strconv"
	"strings"
	"time"

	//	"bufio"
	//	"os"

	//	"runtime"

	"encoding/json"

	//	"html/template"
	"net/http"

	//	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	//	"github.com/PuerkitoBio/goquery"

	//	svg "github.com/ajstarks/svgo/float"

	"github.com/dustin/go-humanize"

	//	exsrapi "github.com/Chouette2100/exsrapi"
	//	srapi "github.com/Chouette2100/srapi"
)

/*

	Ver.00AA00 新規作成（再構成）

*/

const Version = "00AA00"


func GetAciveFanByAPI(room_id string, yyyymm string) (nofan int) {

	nofan = -1

	url := "https://www.showroom-live.com/api/active_fan/room?room_id=" + room_id + "&ym=" + yyyymm

	resp, err := http.Get(url)
	if err != nil {
		//	一時的にデータが取得できない。
		//	resp.Body.Close()
		//		panic(err)
		nofan = -1
		return
	}
	defer resp.Body.Close()

	//	JSONをデコードする。
	//	次の記事を参考にさせていただいております。
	//		Go言語でJSONに泣かないためのコーディングパターン
	//		https://qiita.com/msh5/items/dc524e38073ed8e3831b

	var result interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		//	panic(err)
		nofan = -2
		return
	}

	value, _ := result.(map[string]interface{})["total_user_count"].(float64)
	nofan = int(value)

	return
}


func GetRoomInfoByAPI(room_id string) (
	genre string,
	rank string,
	nrank string,
	prank string,
	level int,
	followers int,
	fans int,
	fans_lst int,
	roomname string,
	roomurlkey string,
	startedat time.Time,
	status int,
) {

	status = 0

	//	https://qiita.com/takeru7584/items/f4ba4c31551204279ed2
	url := "https://www.showroom-live.com/api/room/profile?room_id=" + room_id

	resp, err := http.Get(url)
	if err != nil {
		//	一時的にデータが取得できない。
		//	resp.Body.Close()
		//		panic(err)
		status = -1
		return
	}
	defer resp.Body.Close()

	//	JSONをデコードする。
	//	次の記事を参考にさせていただいております。
	//		Go言語でJSONに泣かないためのコーディングパターン
	//		https://qiita.com/msh5/items/dc524e38073ed8e3831b

	var result interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		//	panic(err)
		status = -2
		return
	}

	value, _ := result.(map[string]interface{})["follower_num"].(float64)
	followers = int(value)

	tnow := time.Now()
	fans = GetAciveFanByAPI(room_id, tnow.Format("200601"))
	yy := tnow.Year()
	mm := tnow.Month() - 1
	if mm < 0 {
		yy -= 1
		mm = 12
	}
	fans_lst = GetAciveFanByAPI(room_id, fmt.Sprintf("%04d%02d", yy, mm))

	genre, _ = result.(map[string]interface{})["genre_name"].(string)

	rank, _ = result.(map[string]interface{})["league_label"].(string)
	ranks, _ := result.(map[string]interface{})["show_rank_subdivided"].(string)
	rank = rank + " | " + ranks

	value, _ = result.(map[string]interface{})["next_score"].(float64)
	nrank = humanize.Comma(int64(value))
	value, _ = result.(map[string]interface{})["prev_score"].(float64)
	prank = humanize.Comma(int64(value))

	value, _ = result.(map[string]interface{})["room_level"].(float64)
	level = int(value)

	roomname, _ = result.(map[string]interface{})["room_name"].(string)

	roomurlkey, _ = result.(map[string]interface{})["room_url_key"].(string)

	//	配信開始時刻の取得
	value, _ = result.(map[string]interface{})["current_live_started_at"].(float64)
	startedat = time.Unix(int64(value), 0).Truncate(time.Second)
	//	log.Printf("current_live_stared_at %f %v\n", value, startedat)

	return

}

// idで指定した配信者さんの獲得ポイントを取得する。
// 戻り値は 獲得ポイント、順位、上位とのポイント差（1位の場合は2位とのポイント差）、イベント名
// レベルイベントのときは順位、上位とのポイント差は0がセットされる。
func GetPointsByAPI(id string) (Point, Rank, Gap int, EventID string) {

	//	獲得ポイントなどの配信者情報を得るURL（このURLについては記事参照）
	URL := "https://www.showroom-live.com/api/room/event_and_support?room_id=" + id

	resp, err := http.Get(URL)
	if err != nil {
		//	一時的にデータが取得できない。
		//		panic(err)
		return 0, 0, 0, "**Error** http.Get(URL)"
	}
	defer resp.Body.Close()

	//	JSONをデコードする。
	//	次の記事を参考にさせていただいております。
	//		Go言語でJSONに泣かないためのコーディングパターン
	//		https://qiita.com/msh5/items/dc524e38073ed8e3831b

	var result interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		//	panic(err)
		return 0, 0, 0, "**Error** http.Get(URL)"
	}

	//	イベントが終わっている、イベント参加をとりやめた、SHOWROOMをやめた、などの対応
	if result.(map[string]interface{})["event"] == nil {
		return 0, 0, 0, "not held yet./over./not entry."
	}

	if result.(map[string]interface{})["event"].(map[string]interface{})["ranking"] != nil {
		//	ランキングのあるイベントの場合
		//	（順位に応じて特典が与えられるイベント、ただし獲得ポイントに対して特典が与えられる場合でも順位付けがある場合はこちら）

		//	獲得ポイント
		l, _ := result.(map[string]interface{})["event"].(map[string]interface{})["ranking"].(map[string]interface{})["point"].(float64)
		//	順位
		m, _ := result.(map[string]interface{})["event"].(map[string]interface{})["ranking"].(map[string]interface{})["rank"].(float64)
		//	ポイント差
		n, _ := result.(map[string]interface{})["event"].(map[string]interface{})["ranking"].(map[string]interface{})["gap"].(float64)

		Point = int(l)
		Rank = int(m)
		Gap = int(n)

		//	イベント名
		EventID, _ = result.(map[string]interface{})["event"].(map[string]interface{})["event_url"].(string)
		EventID = strings.Replace(EventID, "https://www.showroom-live.com/event/", "", -1)

	} else if result.(map[string]interface{})["event"].(map[string]interface{})["quest"] != nil {
		//	レベルイベント（ランキングのないイベント）の場合
		//	（アバ権やステッカーなど獲得ポイントに応じて特典が与えられるイベント、ただし順位付けがある場合は除く）

		//	獲得ポイント
		l, _ := result.(map[string]interface{})["event"].(map[string]interface{})["quest"].(map[string]interface{})["support"].(map[string]interface{})["current_point"].(float64)
		//	順位
		m := 0.0
		//	ポイント差
		n := 0.0

		Point = int(l)
		Rank = int(m)
		Gap = int(n)

		//	イベント名
		EventID, _ = result.(map[string]interface{})["event"].(map[string]interface{})["event_url"].(string)
		EventID = strings.Replace(EventID, "https://www.showroom-live.com/event/", "", -1)

	} else {
		//	上記ランキングイベントでもレベルイベントでもない場合
		//	もしこのようなケースが存在するならJSONを確認して新たにコーディングする
		log.Println(" N/A")
		return 0, 0, 0, "N/A"
	}

	return
}
