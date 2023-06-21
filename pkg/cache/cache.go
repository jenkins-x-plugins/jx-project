package cache

import (
	"fmt"
	"os"
	"time"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

const (
	timeLayout                = time.RFC1123
	defaultFileWritePermisons = 0644

	// TODO make this configurable?
	defaultCacheTimeoutHours = 24
)

// Loader defines cache value population callback that should be executed if cache entry with given key is
// not present.
type Loader func() ([]byte, error)

// LoadCacheData loads cached data from the given cache file name and loader
func LoadCacheData(fileName string, loader Loader) ([]byte, error) {
	if fileName == "" {
		return loader()
	}
	timecheckFileName := fileName + "_last_time_check"
	exists, _ := files.FileExists(fileName)
	if exists {
		// let's check if we should use cache
		if shouldUseCache(timecheckFileName) {
			return os.ReadFile(fileName)
		}
	}
	data, err := loader()
	if err != nil {
		return nil, err
	}

	err2 := os.WriteFile(fileName, data, defaultFileWritePermisons)
	if err2 != nil {
		log.Logger().Warnf("Failed to update cache file %s due to %s", fileName, err2)
	}
	err = writeTimeToFile(timecheckFileName, time.Now())
	if err != nil {
		return nil, err
	}

	return data, err
}

// shouldUseCache returns true if we should use the cached data to serve up the content
func shouldUseCache(filePath string) bool {
	lastUpdateTime := getTimeFromFileIfExists(filePath)
	return time.Since(lastUpdateTime).Hours() < defaultCacheTimeoutHours
}

func writeTimeToFile(path string, inputTime time.Time) error {
	err := os.WriteFile(path, []byte(inputTime.Format(timeLayout)), defaultFileWritePermisons)
	if err != nil {
		return fmt.Errorf("error writing current update time to file: %s", err)
	}
	return nil
}

func getTimeFromFileIfExists(path string) time.Time {
	lastUpdateCheckTime, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	timeInFile, err := time.Parse(timeLayout, string(lastUpdateCheckTime))
	if err != nil {
		return time.Time{}
	}
	return timeInFile
}
