package model

import "time"

// q:  what in tarnation are the `IsNew`s for?
// a:  it's a bit of a hack - but we set a models IsNew to true if
//     we just filled it in for the first time, so when it comes
//     time to insert them (post children callback) we can check for
//     that bool being true - since it won't be true if it was already
//     in the db

// Album represents the albums table
type Album struct {
	IDBase
	CrudBase
	AlbumArtist   AlbumArtist
	AlbumArtistID int    `gorm:"index" sql:"default: null; type:int REFERENCES album_artists(id) ON DELETE CASCADE"`
	Title         string `gorm:"not null; index"`
	// an Album having a `Path` is a little weird when browsing by tags
	// (for the most part - the library's folder structure is treated as
	// if it were flat), but this solves the "American Football problem"
	// https://en.wikipedia.org/wiki/American_Football_(band)#Discography
	Path    string `gorm:"not null; unique_index"`
	CoverID int    `sql:"default: null; type:int REFERENCES covers(id)"`
	Cover   Cover
	Year    int
	Tracks  []Track
	IsNew   bool `gorm:"-"`
}

// AlbumArtist represents the AlbumArtists table
type AlbumArtist struct {
	IDBase
	CrudBase
	Name   string `gorm:"not null; unique_index"`
	Albums []Album
}

// Track represents the tracks table
type Track struct {
	IDBase
	CrudBase
	Album         Album
	AlbumID       int `gorm:"index" sql:"default: null; type:int REFERENCES albums(id) ON DELETE CASCADE"`
	AlbumArtist   AlbumArtist
	AlbumArtistID int `gorm:"index" sql:"default: null; type:int REFERENCES album_artists(id) ON DELETE CASCADE"`
	Artist        string
	Bitrate       int
	Codec         string
	DiscNumber    int
	Duration      int
	Title         string
	TotalDiscs    int
	TotalTracks   int
	TrackNumber   int
	Year          int
	Suffix        string
	ContentType   string
	Size          int
	Folder        Folder
	FolderID      int    `gorm:"not null; index" sql:"default: null; type:int REFERENCES folders(id) ON DELETE CASCADE"`
	Path          string `gorm:"not null; unique_index"`
}

// Cover represents the covers table
type Cover struct {
	IDBase
	CrudBase
	Image []byte
	Path  string `gorm:"not null; unique_index"`
	IsNew bool   `gorm:"-"`
}

// User represents the users table
type User struct {
	IDBase
	CrudBase
	Name          string `gorm:"not null; unique_index"`
	Password      string
	LastFMSession string
	IsAdmin       bool
}

// Setting represents the settings table
type Setting struct {
	CrudBase
	Key   string `gorm:"primary_key; auto_increment:false"`
	Value string
}

// Play represents the settings table
type Play struct {
	IDBase
	User     User
	UserID   int `gorm:"not null; index" sql:"default: null; type:int REFERENCES users(id) ON DELETE CASCADE"`
	Album    Album
	AlbumID  int `gorm:"not null; index" sql:"default: null; type:int REFERENCES albums(id) ON DELETE CASCADE"`
	Folder   Folder
	FolderID int `gorm:"not null; index" sql:"default: null; type:int REFERENCES folders(id) ON DELETE CASCADE"`
	Time     time.Time
	Count    int
}

// Folder represents the settings table
type Folder struct {
	IDBase
	CrudBase
	Name      string
	Path      string `gorm:"not null; unique_index"`
	Parent    *Folder
	ParentID  int  `sql:"default: null; type:int REFERENCES folders(id) ON DELETE CASCADE"`
	CoverID   int  `sql:"default: null; type:int REFERENCES covers(id)"`
	HasTracks bool `gorm:"not null; index"`
	Cover     Cover
	IsNew     bool `gorm:"-"`
}
