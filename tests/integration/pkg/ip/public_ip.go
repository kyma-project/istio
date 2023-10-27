package ip

import (
	"io"
	"log"
	"net/http"
)

// FetchPublic returns the public IP address of the caller by using the ipify API.
func FetchPublic() (string, error) {
	url := "https://api.ipify.org?format=text"
	log.Printf("Getting IP address from  ipify ...\n")
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v\n", err)
		}
	}()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}
