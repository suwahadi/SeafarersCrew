package seeders

import (
	"math/rand/v2"
	"strings"
)

const (
	nationalityIndonesian = "Indonesian"
	nationalityFilipino   = "Filipino"
	nationalityIndian     = "Indian"
	nationalityUkrainian  = "Ukrainian"
	nationalityBurmese    = "Burmese"
)

type demoPerson struct {
	FullName    string
	Gender      string
	Nationality string
	Religion    string
	City        string
	Province    string
	Country     string
	Airport     string
}

type placeInfo struct {
	city, province, country, airport string
}

type namePool struct {
	male, female, last []string
	places             []placeInfo
	religions          []string
}

var namePools = map[string]namePool{
	nationalityIndonesian: {
		male: []string{"Agus", "Budi", "Dedi", "Eko", "Fajar", "Hendra", "Irwan", "Joko", "Rudi", "Yusuf", "Bambang", "Ahmad", "Rizky", "Slamet", "Wahyu",
			"Andi", "Dwi", "Taufik", "Mulyadi", "Syahrul", "Arif", "Bayu", "Candra", "Dimas", "Faisal", "Gunawan", "Heri", "Iwan", "Lukman", "Nanda",
			"Prasetyo", "Rahmat", "Surya", "Teguh", "Yoga", "Zainal", "Hasan", "Ismail", "Samuel", "Yohanes", "Markus", "Daniel", "Ricky", "Stefanus"},
		female: []string{"Sri", "Dewi", "Ratna", "Siti", "Wulan", "Ayu", "Putri", "Indah", "Rina", "Lestari", "Maya", "Fitri"},
		last: []string{"Santoso", "Wijaya", "Pratama", "Saputra", "Hidayat", "Kusuma", "Nugroho", "Setiawan", "Siregar", "Lubis", "Nasution", "Simanjuntak",
			"Hutapea", "Manurung", "Tampubolon", "Wibowo", "Susanto", "Rahman", "Hakim", "Purnomo", "Ramadhan", "Firmansyah", "Halim", "Kurniawan",
			"Permana", "Putra", "Utama", "Yulianto", "Pangaribuan", "Sitompul", "Tambunan", "Rumapea", "Pattiasina", "Latuconsina", "Tuhumury", "Kaligis",
			"Mamahit", "Tumbelaka", "Waworuntu", "Hartono", "Suryadi", "Maulana", "Hermawan", "Budiman", "Lesmana", "Sihombing"},
		places: []placeInfo{
			{"Surabaya", "East Java", "Indonesia", "SUB"}, {"Semarang", "Central Java", "Indonesia", "SRG"}, {"Jakarta", "DKI Jakarta", "Indonesia", "CGK"},
			{"Makassar", "South Sulawesi", "Indonesia", "UPG"}, {"Medan", "North Sumatra", "Indonesia", "KNO"}, {"Balikpapan", "East Kalimantan", "Indonesia", "BPN"},
			{"Batam", "Riau Islands", "Indonesia", "BTH"}, {"Manado", "North Sulawesi", "Indonesia", "MDC"}, {"Bandung", "West Java", "Indonesia", "BDO"},
			{"Cilacap", "Central Java", "Indonesia", "CGK"}, {"Palembang", "South Sumatra", "Indonesia", "PLM"}, {"Banjarmasin", "South Kalimantan", "Indonesia", "BDJ"},
			{"Ambon", "Maluku", "Indonesia", "AMQ"}, {"Tegal", "Central Java", "Indonesia", "SRG"}, {"Lampung", "Lampung", "Indonesia", "TKG"},
		},
		religions: []string{"Islam", "Islam", "Islam", "Islam", "Islam", "Islam", "Islam", "Islam", "Christian", "Christian", "Catholic", "Hindu", "Buddhist"},
	},
	nationalityFilipino: {
		male:      []string{"Jose", "Mark", "Ramon", "Rolando", "Arnel", "Ronnie", "Jericho", "Noel", "Allan", "Dennis"},
		female:    []string{"Maria", "Grace", "Liza", "Joy"},
		last:      []string{"Santos", "Reyes", "Cruz", "Dela Cruz", "Bautista", "Ramos", "Mendoza", "Aquino", "Villanueva", "Garcia"},
		places:    []placeInfo{{"Manila", "Metro Manila", "Philippines", "MNL"}, {"Cebu", "Cebu", "Philippines", "CEB"}, {"Iloilo", "Iloilo", "Philippines", "ILO"}},
		religions: []string{"Catholic"},
	},
	nationalityIndian: {
		male:      []string{"Rajesh", "Sandeep", "Vikram", "Anil", "Suresh", "Manoj", "Prakash", "Deepak"},
		female:    []string{"Anita", "Priya"},
		last:      []string{"Sharma", "Verma", "Nair", "Pillai", "Reddy", "Singh", "Menon", "Iyer"},
		places:    []placeInfo{{"Mumbai", "Maharashtra", "India", "BOM"}, {"Kochi", "Kerala", "India", "COK"}, {"Chennai", "Tamil Nadu", "India", "MAA"}},
		religions: []string{"Hindu", "Hindu", "Christian"},
	},
	nationalityUkrainian: {
		male:      []string{"Oleksandr", "Mykola", "Andriy", "Taras", "Serhii", "Volodymyr"},
		female:    []string{"Olena", "Iryna"},
		last:      []string{"Kovalenko", "Shevchenko", "Bondarenko", "Tkachenko", "Melnyk", "Kravchenko"},
		places:    []placeInfo{{"Odesa", "Odesa Oblast", "Ukraine", "ODS"}, {"Kherson", "Kherson Oblast", "Ukraine", "ODS"}},
		religions: []string{"Orthodox"},
	},
	nationalityBurmese: {
		male:      []string{"Aung", "Kyaw", "Zaw", "Htet", "Min"},
		female:    []string{"Khin", "Thida"},
		last:      []string{"Win", "Naing", "Myint", "Oo", "Lwin", "Soe"},
		places:    []placeInfo{{"Yangon", "Yangon Region", "Myanmar", "RGN"}, {"Mawlamyine", "Mon State", "Myanmar", "RGN"}},
		religions: []string{"Buddhist"},
	},
}

func pick[T any](rng *rand.Rand, items []T) T {
	return items[rng.IntN(len(items))]
}

func newPerson(rng *rand.Rand, nationality, gender string, taken map[string]bool) demoPerson {
	pool := namePools[nationality]
	firsts := pool.male
	if gender == "female" {
		firsts = pool.female
	}
	for attempt := 0; ; attempt++ {
		name := pick(rng, firsts) + " " + pick(rng, pool.last)
		if attempt > 40 {
			name += " " + pick(rng, pool.last)
		}
		if taken[strings.ToLower(name)] {
			continue
		}
		taken[strings.ToLower(name)] = true
		place := pick(rng, pool.places)
		return demoPerson{
			FullName: name, Gender: gender, Nationality: nationality, Religion: pick(rng, pool.religions),
			City: place.city, Province: place.province, Country: place.country, Airport: place.airport,
		}
	}
}
