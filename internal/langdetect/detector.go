package langdetect

import (
"strings"
"unicode"

"github.com/abadojack/whatlanggo"
)

var TargetLanguage = map[string]whatlanggo.Lang{
"english":            whatlanggo.Eng,
"en":                 whatlanggo.Eng,
"chinese":            whatlanggo.Cmn,
"mandarin":           whatlanggo.Cmn,
"traditional chinese": whatlanggo.Cmn,
"simplified chinese":  whatlanggo.Cmn,
"japanese":           whatlanggo.Jpn,
"korean":             whatlanggo.Kor,
"russian":            whatlanggo.Rus,
"spanish":            whatlanggo.Spa,
"french":             whatlanggo.Fra,
"german":             whatlanggo.Deu,
}

type Detector struct{}

func NewDetector() *Detector {
return &Detector{}
}

func (d *Detector) Detect(text string) (lang string, confidence float64, script string) {
info := whatlanggo.Detect(text)
return info.Lang.String(), info.Confidence, whatlanggo.Scripts[info.Script]
}

func (d *Detector) IsTargetLanguage(text string, targetLang string) bool {
targetLower := strings.ToLower(targetLang)

if strings.Contains(targetLower, "chinese") || targetLower == "zh" || targetLower == "mandarin" {
info := whatlanggo.Detect(text)
return info.Script == unicode.Han || info.Lang == whatlanggo.Cmn
}

expected, ok := TargetLanguage[targetLower]
if !ok {
return true
}

info := whatlanggo.Detect(text)
return info.Lang == expected || info.Confidence < 0.5
}

func (d *Detector) GetSourceLanguage(texts []string) string {
combined := strings.Join(texts, " ")
info := whatlanggo.Detect(combined)
return info.Lang.String()
}
