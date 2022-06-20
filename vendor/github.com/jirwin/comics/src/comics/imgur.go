package comics

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	scan "github.com/mattn/go-scan"
)

const endpoint = "https://api.imgur.com/3/image"

func ImgurUpload(img []byte, clientID string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(img)
	params := url.Values{"image": {encoded}}

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "post:", err.Error())
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Client-ID "+clientID)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		var message string
		err = scan.ScanJSON(res.Body, "data/error", &message)
		if err != nil {
			message = res.Status
		}
		return "", fmt.Errorf("%s", message)
	}
	defer res.Body.Close()

	var link string
	err = scan.ScanJSON(res.Body, "data/link", &link)
	if err != nil {
		return "", err
	}

	return link, nil
}
