package main

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

func downloadImageForMac(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", errors.New("non-200 status code")
	}

	file, err := ioutil.TempFile("", "walltakerbg")
	if err != nil {
		return "", err
	}

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return "", err
	}

	err = file.Close()
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}

func cleanUpCacheForMac(file string) error {
	return os.Remove(file)
}
