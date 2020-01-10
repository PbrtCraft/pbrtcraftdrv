package mc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var (
	uuidNameMap map[string]string
)

func init() {
	if err := readUUIDCache(); err != nil {
		uuidNameMap = map[string]string{}
	}
}

func readUUIDCache() error {
	bs, err := ioutil.ReadFile("usercache.json")
	if err != nil {
		return fmt.Errorf("app.main.readUUIDCache: %s", err)
	}
	err = json.Unmarshal(bs, &uuidNameMap)
	if err != nil {
		return fmt.Errorf("app.main.readUUIDCache: %s", err)
	}
	return nil
}

func writeUUIDCache() error {
	bs, err := json.Marshal(uuidNameMap)
	if err != nil {
		return fmt.Errorf("app.main.writeUUIDCache: %s", err)
	}
	err = ioutil.WriteFile("usercache.json", bs, 0666)
	if err != nil {
		return fmt.Errorf("app.main.writeUUIDCache: %s", err)
	}
	return nil
}

func uuidToName(uuid string) (string, error) {
	if name, exist := uuidNameMap[uuid]; exist {
		return name, nil
	}

	url := fmt.Sprintf("https://api.mojang.com/user/profiles/%s/names",
		strings.Replace(uuid, "-", "", -1))
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("app.uuidToName: %s", err)
	}
	req.Header.Set("User-Agent", "Get Name By UUID")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("app.uuidToName: %s", err)
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("app.uuidToName: %s", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("app.uuidToName: Request StatusCode = %d, %s",
			resp.StatusCode, string(bytes))
	}

	var tmp []struct {
		Name string `json:"name"`
	}
	err = json.Unmarshal(bytes, &tmp)
	if err != nil {
		return "", fmt.Errorf("app.uuidToName: %s", err)
	}
	if len(tmp) == 0 {
		return "", fmt.Errorf("app.uuidToName: cannot found player by uuid %s", uuid)
	}

	uuidNameMap[uuid] = tmp[0].Name

	err = writeUUIDCache()
	if err != nil {
		log.Println(err)
	}

	return tmp[0].Name, nil
}
