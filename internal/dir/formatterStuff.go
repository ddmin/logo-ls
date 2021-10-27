package dir

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ddmin/logo-ls/assets"
	"github.com/ddmin/logo-ls/internal/api"
)

func mainSort(a, b string) bool {
	switch a {
	case ".", "..":
	default:
		a = strings.TrimPrefix(a, ".")
	}
	switch b {
	case ".", "..":
	default:
		b = strings.TrimPrefix(b, ".")
	}
	return strings.ToLower(a) < strings.ToLower(b)
}

// Custom less functions
func lessFuncGenerator(d *dir) {
	switch {
	case (api.FlagVector & api.Flag_alpha) > 0:
		// sort by alphabetical order of name.ext
		d.less = func(i, j int) bool {
			return mainSort(d.files[i].name+d.files[i].ext, d.files[j].name+d.files[j].ext)
		}
	case (api.FlagVector & api.Flag_S) > 0:
		// sort by file size, largest first
		d.less = func(i, j int) bool {
			if d.files[i].size > d.files[j].size {
				return true
			} else if d.files[i].size == d.files[j].size {
				return mainSort(d.files[i].name+d.files[i].ext, d.files[j].name+d.files[j].ext)
			} else {
				return false
			}
		}
	case (api.FlagVector & api.Flag_t) > 0:
		// sort by modification time, newest first
		// not sorting by alphabetical order because equality is quite rare
		d.less = func(i, j int) bool {
			return d.files[i].modTime.After(d.files[j].modTime)
		}
	case (api.FlagVector & api.Flag_X) > 0:
		// sort alphabetically by entry extension
		d.less = func(i, j int) bool {
			if mainSort(d.files[i].ext, d.files[j].ext) {
				return true
			} else if strings.ToLower(d.files[i].ext) == strings.ToLower(d.files[j].ext) {
				return mainSort(d.files[i].name+d.files[i].ext, d.files[j].name+d.files[j].ext)
			} else {
				return false
			}
		}
	case (api.FlagVector & api.Flag_v) > 0:
		// natural sort of (version) numbers within text
		d.less = func(i, j int) bool {
			// compare if files are directories
			dirI := (d.files[i].indicator == "/")
			dirJ := (d.files[j].indicator == "/")

			// group directories first
			if dirI && dirJ {
				// hidden files are weird
				if d.files[i].name == d.files[j].name {
					return d.files[i].ext < d.files[j].ext
				}

                // re pattern to remove certain characters
                str_re := regexp.MustCompile(`\d+`)
                int_re := regexp.MustCompile(`\D+`)

                // remove all letters and non-digit characters
                intI, iErr := strconv.Atoi(string(int_re.ReplaceAll([]byte(d.files[i].name), []byte{})))
                intJ, jErr := strconv.Atoi(string(int_re.ReplaceAll([]byte(d.files[j].name), []byte{})))

                if iErr != nil || jErr != nil {
                    return d.files[i].name < d.files[j].name
                }

                // remove all letters and non-digit characters
                strI := string(str_re.ReplaceAll([]byte(d.files[i].name), []byte{}))
                strJ := string(str_re.ReplaceAll([]byte(d.files[j].name), []byte{}))

                if strI != strJ {
                    return d.files[i].name < d.files[j].name
                }
                return intI < intJ

			} else if dirI && !dirJ {
				return true
			} else if !dirI && dirJ {
				return false
			}

			// check if executable
			exI := (d.files[i].indicator == "*")
			exJ := (d.files[j].indicator == "*")

			// group executables last
			if exI && exJ {
				return d.files[i].name < d.files[j].name
			} else if exI && !exJ {
				return false
			} else if !exI && exJ {
				return true
			}

			// check if symlink
			if strings.Contains(d.files[i].ext, "->") || strings.Contains(d.files[j].ext, "->") {
				return d.files[i].name < d.files[j].name
			}

			// compare by extension
			if d.files[i].ext != d.files[j].ext {
				return d.files[i].ext < d.files[j].ext
			}

			// re pattern to remove certain characters
			str_re := regexp.MustCompile(`\d+`)
			int_re := regexp.MustCompile(`\D+`)

			// remove all letters and non-digit characters
			intI, iErr := strconv.Atoi(string(int_re.ReplaceAll([]byte(d.files[i].name), []byte{})))
			intJ, jErr := strconv.Atoi(string(int_re.ReplaceAll([]byte(d.files[j].name), []byte{})))

			if iErr != nil || jErr != nil {
				return d.files[i].name < d.files[j].name
			}

			// remove all letters and non-digit characters
			strI := string(str_re.ReplaceAll([]byte(d.files[i].name), []byte{}))
			strJ := string(str_re.ReplaceAll([]byte(d.files[j].name), []byte{}))

			if strI != strJ {
				return d.files[i].name < d.files[j].name
			}
			return intI < intJ
		}
	default:
		// no sorting
		d.less = func(i, j int) bool {
			return i < j
		}
	}
}

// get Owner and Group info
var grpMap = make(map[string]string)
var userMap = make(map[string]string)

// get indicator of the file
func getIndicator(modebit os.FileMode) (i string) {
	if modebit&os.ModeDir > 0 {
		i = "/"
	} else if modebit&os.ModeNamedPipe > 0 {
		i = "|"
	} else if modebit&os.ModeSymlink > 0 {
		i = ""
	} else if modebit&os.ModeSocket > 0 {
		i = "="
	} else if modebit&1000000 > 0 {
		i = "*"
	}
	return i
}

func getSizeInFormate(b int64) string {
	if api.FlagVector&api.Flag_h == 0 {
		return fmt.Sprintf("%d", b)
	}

	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func getIcon(name, ext, indicator string) (icon, color string) {
	var i *assets.Icon_Info
	var ok bool

	switch indicator {
	case "/":
		i, ok = assets.Icon_Dir[strings.ToLower(name+ext)]
		if ok {
			break
		}
		if len(name) == 0 || '.' == name[0] {
			i = assets.Icon_Def["hiddendir"]
			break
		}
		i = assets.Icon_Def["dir"]
	default:
		i, ok = assets.Icon_FileName[strings.ToLower(name+ext)]
		if ok {
			break
		}

		// a special admiration for goLang
		if ext == ".go" && strings.HasSuffix(name, "_test") {
			i = assets.Icon_Set["go-test"]
			break
		}

		t := strings.Split(name, ".")
		if len(t) > 1 && t[0] != "" {
			i, ok = assets.Icon_SubExt[strings.ToLower(t[len(t)-1]+ext)]
			if ok {
				break
			}
		}

		i, ok = assets.Icon_Ext[strings.ToLower(strings.TrimPrefix(ext, "."))]
		if ok {
			break
		}

		if len(name) == 0 || '.' == name[0] {
			i = assets.Icon_Def["hiddenfile"]
			break
		}
		i = assets.Icon_Def["file"]
	}

	// change icon color if the file is executable
	if indicator == "*" {
		if i.GetGlyph() == "\uf723" {
			i = assets.Icon_Def["exe"]
		}
		i.MakeExe()
	}

	return i.GetGlyph(), i.GetColor(1)
}
