package scd

type SongData struct {
	Title     string
	Author    string
	Url       string
	Available bool
}

type PlaylistData struct {
	Title      string
	Author     string
	Url        string
	TrackCount int
}

type AlbumData struct {
	Title      string
	Author     string
	Url        string
	TrackCount int
}

type FetchResponse struct {
	data  []byte
	index int
}
