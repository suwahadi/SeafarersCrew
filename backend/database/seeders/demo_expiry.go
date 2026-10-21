package seeders

import (
	"math/rand/v2"
	"time"
)

type expiryBucket int

const (
	bucketExpired expiryBucket = iota
	bucketWithin30
	bucketWithin60
	bucketWithin90
	bucketValid
)

type bucketWeight struct {
	bucket expiryBucket
	weight int
}

var expiryWeights = []bucketWeight{
	{bucketExpired, 7},
	{bucketWithin30, 6},
	{bucketWithin60, 6},
	{bucketWithin90, 6},
	{bucketValid, 75},
}

func pickExpiryBucket(rng *rand.Rand) expiryBucket {
	total := 0
	for _, w := range expiryWeights {
		total += w.weight
	}
	n := rng.IntN(total)
	for _, w := range expiryWeights {
		if n < w.weight {
			return w.bucket
		}
		n -= w.weight
	}
	return bucketValid
}

type datedItem struct {
	issued time.Time
	expiry time.Time
}

const minValidDays = 91

func expiryFor(rng *rand.Rand, bucket expiryBucket, today time.Time, validityDays int) datedItem {
	var daysFromToday int
	switch bucket {
	case bucketExpired:
		daysFromToday = -(1 + rng.IntN(180))
	case bucketWithin30:
		daysFromToday = 1 + rng.IntN(30)
	case bucketWithin60:
		daysFromToday = 31 + rng.IntN(30)
	case bucketWithin90:
		daysFromToday = 61 + rng.IntN(30)
	default:
		daysFromToday = minValidDays + rng.IntN(max(validityDays-minValidDays, 1))
	}
	expiry := today.AddDate(0, 0, daysFromToday)
	return datedItem{issued: expiry.AddDate(0, 0, -validityDays), expiry: expiry}
}

func bucketOf(expiry, today time.Time) expiryBucket {
	days := int(expiry.Sub(today).Hours() / 24)
	switch {
	case days < 0:
		return bucketExpired
	case days <= 30:
		return bucketWithin30
	case days <= 60:
		return bucketWithin60
	case days <= 90:
		return bucketWithin90
	default:
		return bucketValid
	}
}
