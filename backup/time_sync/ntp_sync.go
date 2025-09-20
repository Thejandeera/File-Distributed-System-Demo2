package time_sync

import (
	"time"

	"log"

	"github.com/beevik/ntp"
)

var ClockSkew time.Duration

func SyncClock() {
	ntpServer := "0.beevik-ntp.pool.ntp.org"

	for {
		time.Sleep(30 * time.Second)

		resp, err := ntp.Query(ntpServer)
		if err != nil {
			log.Println("❌ Failed to sync NTP:", err)
			continue
		}

		ClockSkew = resp.ClockOffset
		log.Println("⏰ Synced clock skew:", ClockSkew)
	}
}

func GetCorrectedTime() time.Time {
	return time.Now().Add(ClockSkew)
}
