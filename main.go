package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// СЛОВАРЬ ЛОКАЛИЗАЦИИ
// ============================================================================

var locales = map[string]map[string]string{
	"ru": {
		"LangName":      "EN",
		"LangLink":      "/lang?l=en",
		"NavGuest":      "Гость",
		"NavRecipes":    "Рецепты",
		"NavLogin":      "Войти",
		"NavRegister":   "Регистрация",
		"NavLogout":     "Выйти",
		"NavProfile":    "Личный кабинет",
		"TitleLogin":    "Вход",
		"TitleReg":      "Регистрация",
		"TitleGuest":    "Гостевой расчет (Без сохранения)",
		"TitlePersonal": "Персональный расчет",
		"TitleHistory":  "История сессий",
		"FieldUser":     "Логин",
		"FieldPass":     "Пароль",
		"FieldWeight":   "Вес (кг)",
		"FieldHeight":   "Рост (см)",
		"FieldGender":   "Пол",
		"GenderM":       "Мужской",
		"GenderF":       "Женский",
		"FieldTHC":      "ТГК %",
		"FieldProduct":  "Тип стаффа",
		"ProdBuds":      "Шишки",
		"ProdHash":      "Гашиш",
		"ProdTrim":      "Рассыпуха",
		"FieldGenetics": "Генетика",
		"GenHybrid":     "Гибрид",
		"GenSativa":     "Сатива",
		"GenIndica":     "Индика",
		"FieldIntens":   "Уровень накура",
		"IntMicro":      "Микродозинг",
		"IntNormal":     "Стандарт",
		"IntStrong":     "Плотный",
		"IntExtreme":    "Экстрим",
		"FieldMethod":   "Девайс",
		"MethJoint":     "Косяк",
		"MethBong":      "Бонг / Водный",
		"MethVape":      "Вапорайзер",
		"MethEdible":    "Еда",
		"BtnLogin":      "Войти",
		"BtnReg":        "Создать профиль",
		"BtnCalc":       "Считать",
		"BtnCalcSave":   "Рассчитать и сохранить",
		"ErrLogin":      "Неверный логин или пароль",
		"ErrReg":        "Логин уже занят",
		"HistDate":      "Дата",
		"HistProduct":   "Продукт",
		"HistMethod":    "Метод",
		"HistDose":      "Доза (г)",
		"HistEmpty":     "У вас пока нет сохраненных расчетов.",
		"MsgSuccess":    "Расчет успешен: <strong>%.2f г</strong>. Результат сохранен в историю.",
		"MsgOptimal":    "Оптимально: %.2f г",
	},
	"en": {
		"LangName":      "RU",
		"LangLink":      "/lang?l=ru",
		"NavGuest":      "Guest",
		"NavRecipes":    "Recipes",
		"NavLogin":      "Login",
		"NavRegister":   "Register",
		"NavLogout":     "Logout",
		"NavProfile":    "Dashboard",
		"TitleLogin":    "Login",
		"TitleReg":      "Registration",
		"TitleGuest":    "Guest Calculation (No Save)",
		"TitlePersonal": "Personal Calculation",
		"TitleHistory":  "Session History",
		"FieldUser":     "Username",
		"FieldPass":     "Password",
		"FieldWeight":   "Weight (kg)",
		"FieldHeight":   "Height (cm)",
		"FieldGender":   "Gender",
		"GenderM":       "Male",
		"GenderF":       "Female",
		"FieldTHC":      "THC %",
		"FieldProduct":  "Product Type",
		"ProdBuds":      "Buds",
		"ProdHash":      "Hash",
		"ProdTrim":      "Trim",
		"FieldGenetics": "Genetics",
		"GenHybrid":     "Hybrid",
		"GenSativa":     "Sativa",
		"GenIndica":     "Indica",
		"FieldIntens":   "Intensity",
		"IntMicro":      "Microdose",
		"IntNormal":     "Standard",
		"IntStrong":     "Strong",
		"IntExtreme":    "Extreme",
		"FieldMethod":   "Method",
		"MethJoint":     "Joint",
		"MethBong":      "Bong",
		"MethVape":      "Vaporizer",
		"MethEdible":    "Edible",
		"BtnLogin":      "Login",
		"BtnReg":        "Create Profile",
		"BtnCalc":       "Calculate",
		"BtnCalcSave":   "Calculate & Save",
		"ErrLogin":      "Invalid username or password",
		"ErrReg":        "Username is already taken",
		"HistDate":      "Date",
		"HistProduct":   "Product",
		"HistMethod":    "Method",
		"HistDose":      "Dose (g)",
		"HistEmpty":     "No calculations saved yet.",
		"MsgSuccess":    "Calculation successful: <strong>%.2f g</strong>. Saved to history.",
		"MsgOptimal":    "Optimal: %.2f g",
	},
}

// ============================================================================
// МОДЕЛИ БАЗЫ ДАННЫХ
// ============================================================================

type User struct {
	ID            uint   `gorm:"primaryKey"`
	Username      string `gorm:"unique"`
	PasswordHash  string
	SessionToken  string `gorm:"index"`
	WeightKg      float64
	HeightCm      float64
	Gender        string
	BaseTol       float64
	LastSession   time.Time
	TotalSessions int
	Histories     []History `gorm:"foreignKey:UserID"`
}

type Strain struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string `gorm:"unique"`
	THCPercent   float64
	StrainType   string
	GrowMethod   string
	IsAutoflower bool
}

type History struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	Date      time.Time
	Strain    string
	Method    string
	Product   string
	DoseGrams float64
}

var db *gorm.DB

var Bioavailability = map[string]float64{
	"Vaporizer": 0.50,
	"Bong":      0.30,
	"Joint":     0.20,
	"Edible":    0.15,
}

// ============================================================================
// БИЗНЕС-ЛОГИКА (КАЛЬКУЛЯТОР)
// ============================================================================

type CalcConfig struct {
	User      *User
	Strain    *Strain
	Method    string
	Product   string
	Intensity float64
}

func CalculateDose(cfg CalcConfig) float64 {
	baseTargetMg := cfg.User.WeightKg * 0.15

	if strings.ToUpper(cfg.User.Gender) == "F" {
		baseTargetMg *= 0.85
	}

	if cfg.User.HeightCm > 100 {
		heightM := cfg.User.HeightCm / 100.0
		bmi := cfg.User.WeightKg / (heightM * heightM)
		if bmi > 25.0 {
			baseTargetMg *= 1.1
		} else if bmi < 18.5 {
			baseTargetMg *= 0.9
		}
	}

	if cfg.User.TotalSessions > 0 {
		daysSinceLast := time.Since(cfg.User.LastSession).Hours() / 24.0
		if daysSinceLast < 0 {
			daysSinceLast = 0
		}
		currentTol := cfg.User.BaseTol * math.Exp(-0.12*daysSinceLast)
		baseTargetMg *= currentTol
	}

	switch t := strings.ToLower(strings.TrimSpace(cfg.Strain.StrainType)); t {
	case "sativa", "сатива":
		baseTargetMg *= 0.95
	case "indica", "индика":
		baseTargetMg *= 1.05
	}

	baseTargetMg *= cfg.Intensity

	effectiveTHCPercent := cfg.Strain.THCPercent
	switch cfg.Product {
	case "Hash":
		effectiveTHCPercent = 45.0
	case "Trim":
		effectiveTHCPercent *= 0.3
	}

	absorptionFactor, exists := Bioavailability[cfg.Method]
	if !exists {
		absorptionFactor = 0.20
	}

	if cfg.Method == "Edible" {
		baseTargetMg *= 0.5
	}

	thcPerGramMg := 1000.0 * (effectiveTHCPercent / 100.0)
	gramsNeeded := baseTargetMg / (thcPerGramMg * absorptionFactor)

	return math.Round(gramsNeeded*100) / 100
}

// ============================================================================
// УТИЛИТЫ
// ============================================================================

func hashPassword(pass string) string {
	h := sha256.New()
	h.Write([]byte(pass))
	return hex.EncodeToString(h.Sum(nil))
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getLang(r *http.Request) string {
	cookie, err := r.Cookie("lang")
	if err == nil && (cookie.Value == "ru" || cookie.Value == "en") {
		return cookie.Value
	}
	return "ru"
}

func getUserFromCookie(r *http.Request) *User {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		return nil
	}
	var user User
	if err := db.Preload("Histories", func(db *gorm.DB) *gorm.DB {
		return db.Order("date DESC")
	}).Where("session_token = ?", cookie.Value).First(&user).Error; err != nil {
		return nil
	}
	return &user
}

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("system.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	db.AutoMigrate(&User{}, &Strain{}, &History{})
}

// ============================================================================
// ШАБЛОНЫ WEB-ИНТЕРФЕЙСА
// ============================================================================

const layoutHTML = `
<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <title>THC Control</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@1/css/pico.min.css">
    <style>body { padding: 20px; max-width: 900px; margin: auto; }</style>
</head>
<body>
    <nav>
        <ul><li><strong>THC Control</strong></li></ul>
        <ul>
            <li><a href="/">{{.T.NavGuest}}</a></li>
            {{if .User}}
                <li><a href="/dashboard">{{.T.NavProfile}} ({{.User.Username}})</a></li>
                <li><a href="/logout">{{.T.NavLogout}}</a></li>
            {{else}}
                <li><a href="/login">{{.T.NavLogin}}</a></li>
                <li><a href="/register">{{.T.NavRegister}}</a></li>
            {{end}}
            <li><a href="{{.T.LangLink}}"><b>[{{.T.LangName}}]</b></a></li>
        </ul>
    </nav>
    <main>
        {{if .Message}}<article style="background: #2d5a27; color: white; padding: 10px;">{{.Message}}</article>{{end}}
        {{if .Error}}<article style="background: #a93226; color: white; padding: 10px;">{{.Error}}</article>{{end}}
        {{.Content}}
    </main>
</body>
</html>
`

const dashboardHTML = `
<h2>{{.T.TitlePersonal}}</h2>
<grid>
    <div>
        <form action="/dashboard/calculate" method="POST">
            <label>{{.T.FieldTHC}} <input type="number" step="0.1" name="thc" value="18.0" required></label>
            <label>{{.T.FieldProduct}}
                <select name="product">
                    <option value="Buds">{{.T.ProdBuds}}</option>
                    <option value="Hash">{{.T.ProdHash}}</option>
                    <option value="Trim">{{.T.ProdTrim}}</option>
                </select>
            </label>
            <label>{{.T.FieldGenetics}}
                <select name="strain_type">
                    <option value="Hybrid">{{.T.GenHybrid}}</option>
                    <option value="Sativa">{{.T.GenSativa}}</option>
                    <option value="Indica">{{.T.GenIndica}}</option>
                </select>
            </label>
            <label>{{.T.FieldIntens}}
                <select name="intensity">
                    <option value="0.5">{{.T.IntMicro}}</option>
                    <option value="1.0" selected>{{.T.IntNormal}}</option>
                    <option value="1.5">{{.T.IntStrong}}</option>
                    <option value="2.0">{{.T.IntExtreme}}</option>
                </select>
            </label>
            <label>{{.T.FieldMethod}}
                <select name="method">
                    <option value="Joint">{{.T.MethJoint}}</option>
                    <option value="Bong">{{.T.MethBong}}</option>
                    <option value="Vaporizer">{{.T.MethVape}}</option>
                    <option value="Edible">{{.T.MethEdible}}</option>
                </select>
            </label>
            <button type="submit">{{.T.BtnCalcSave}}</button>
        </form>
    </div>
    
    <div>
        <h3>{{.T.TitleHistory}}</h3>
        {{if .User.Histories}}
            <figure>
                <table role="grid">
                    <thead><tr><th>{{.T.HistDate}}</th><th>{{.T.HistProduct}}</th><th>{{.T.HistMethod}}</th><th>{{.T.HistDose}}</th></tr></thead>
                    <tbody>
                        {{range .User.Histories}}
                        <tr>
                            <td>{{.Date.Format "02.01.06 15:04"}}</td>
                            <td>{{.Product}} ({{.Strain}})</td>
                            <td>{{.Method}}</td>
                            <td><strong>{{.DoseGrams}}</strong></td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </figure>
        {{else}}
            <p>{{.T.HistEmpty}}</p>
        {{end}}
    </div>
</grid>
`

func renderHTML(w http.ResponseWriter, r *http.Request, content string, data map[string]interface{}) {
	lang := getLang(r)
	if data == nil {
		data = make(map[string]interface{})
	}
	data["User"] = getUserFromCookie(r)
	data["T"] = locales[lang]

	tmplContent, _ := template.New("content").Parse(content)
	var contentBuf strings.Builder
	tmplContent.Execute(&contentBuf, data)
	data["Content"] = template.HTML(contentBuf.String())

	tmplLayout, _ := template.New("layout").Parse(layoutHTML)
	tmplLayout.Execute(w, data)
}

func startWebServer() {
	http.HandleFunc("/lang", func(w http.ResponseWriter, r *http.Request) {
		l := r.URL.Query().Get("l")
		if l == "en" || l == "ru" {
			http.SetCookie(w, &http.Cookie{Name: "lang", Value: l, Path: "/"})
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		form := `
		<h3>{{.T.TitleGuest}}</h3>
		<form action="/" method="POST">
			<grid>
				<label>{{.T.FieldWeight}} <input type="number" name="weight" value="75" required></label>
				<label>{{.T.FieldHeight}} <input type="number" name="height" value="180" required></label>
				<label>{{.T.FieldGender}} <select name="gender"><option value="M">{{.T.GenderM}}</option><option value="F">{{.T.GenderF}}</option></select></label>
			</grid>
			<grid>
				<label>{{.T.FieldTHC}} <input type="number" step="0.1" name="thc" value="18.0" required></label>
				<label>{{.T.FieldMethod}} <select name="method"><option value="Joint">{{.T.MethJoint}}</option><option value="Bong">{{.T.MethBong}}</option><option value="Vaporizer">{{.T.MethVape}}</option></select></label>
			</grid>
			<button type="submit">{{.T.BtnCalc}}</button>
		</form>
		`
		if r.Method == "POST" {
			r.ParseForm()
			wKg, _ := strconv.ParseFloat(r.FormValue("weight"), 64)
			hCm, _ := strconv.ParseFloat(r.FormValue("height"), 64)
			thc, _ := strconv.ParseFloat(r.FormValue("thc"), 64)

			u := &User{WeightKg: wKg, HeightCm: hCm, Gender: r.FormValue("gender")}
			s := &Strain{THCPercent: thc, StrainType: "Hybrid"}
			cfg := CalcConfig{User: u, Strain: s, Method: r.FormValue("method"), Product: "Buds", Intensity: 1.0}

			res := CalculateDose(cfg)
			lang := getLang(r)
			msg := fmt.Sprintf(locales[lang]["MsgOptimal"], res)
			form += fmt.Sprintf(`<article><h2 style="color:#2ecc71">%s</h2></article>`, msg)
		}
		renderHTML(w, r, form, nil)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			username, pass := r.FormValue("username"), r.FormValue("password")

			var user User
			if err := db.Where("username = ? AND password_hash = ?", username, hashPassword(pass)).First(&user).Error; err != nil {
				lang := getLang(r)
				renderHTML(w, r, `<h3>{{.T.TitleLogin}}</h3><form method="POST"><input name="username" placeholder="{{.T.FieldUser}}"><input type="password" name="password" placeholder="{{.T.FieldPass}}"><button>{{.T.BtnLogin}}</button></form>`, map[string]interface{}{"Error": locales[lang]["ErrLogin"]})
				return
			}

			token := generateToken()
			db.Model(&user).Update("session_token", token)
			http.SetCookie(w, &http.Cookie{Name: "session_token", Value: token, Path: "/"})
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		renderHTML(w, r, `<h3>{{.T.TitleLogin}}</h3><form method="POST"><input name="username" placeholder="{{.T.FieldUser}}" required><input type="password" name="password" placeholder="{{.T.FieldPass}}" required><button>{{.T.BtnLogin}}</button></form>`, nil)
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			wKg, _ := strconv.ParseFloat(r.FormValue("weight"), 64)
			hCm, _ := strconv.ParseFloat(r.FormValue("height"), 64)

			user := User{
				Username:     r.FormValue("username"),
				PasswordHash: hashPassword(r.FormValue("password")),
				WeightKg:     wKg,
				HeightCm:     hCm,
				Gender:       r.FormValue("gender"),
				BaseTol:      1.0,
			}

			if err := db.Create(&user).Error; err != nil {
				lang := getLang(r)
				renderHTML(w, r, `<h3>{{.T.TitleReg}}</h3>...`, map[string]interface{}{"Error": locales[lang]["ErrReg"]})
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		form := `
		<h3>{{.T.TitleReg}}</h3>
		<form method="POST">
			<input name="username" placeholder="{{.T.FieldUser}}" required>
			<input type="password" name="password" placeholder="{{.T.FieldPass}}" required>
			<grid>
				<input type="number" name="weight" placeholder="{{.T.FieldWeight}}" required>
				<input type="number" name="height" placeholder="{{.T.FieldHeight}}" required>
				<select name="gender"><option value="M">{{.T.GenderM}}</option><option value="F">{{.T.GenderF}}</option></select>
			</grid>
			<button>{{.T.BtnReg}}</button>
		</form>`
		renderHTML(w, r, form, nil)
	})

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session_token", Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		user := getUserFromCookie(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		renderHTML(w, r, dashboardHTML, nil)
	})

	http.HandleFunc("/dashboard/calculate", func(w http.ResponseWriter, r *http.Request) {
		user := getUserFromCookie(r)
		if user == nil || r.Method != "POST" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		r.ParseForm()
		thc, _ := strconv.ParseFloat(r.FormValue("thc"), 64)
		intensity, _ := strconv.ParseFloat(r.FormValue("intensity"), 64)
		strainType := r.FormValue("strain_type")
		product := r.FormValue("product")
		method := r.FormValue("method")

		strain := &Strain{THCPercent: thc, StrainType: strainType}
		cfg := CalcConfig{User: user, Strain: strain, Method: method, Product: product, Intensity: intensity}

		dose := CalculateDose(cfg)

		db.Create(&History{
			UserID:    user.ID,
			Date:      time.Now(),
			Strain:    strainType,
			Method:    method,
			Product:   product,
			DoseGrams: dose,
		})

		db.Model(user).Updates(map[string]interface{}{
			"total_sessions": user.TotalSessions + 1,
			"last_session":   time.Now(),
		})

		lang := getLang(r)
		msg := fmt.Sprintf(locales[lang]["MsgSuccess"], dose)
		renderHTML(w, r, dashboardHTML, map[string]interface{}{"Message": template.HTML(msg)})
	})

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func startCLI() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\n==============================================")
		fmt.Println("   THC CONTROL (CLI Mode)")
		fmt.Println("==============================================")
		fmt.Println("Use Web Interface for full functionality.")
		fmt.Println("0. Exit")
		fmt.Print("Choice: ")
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) == "0" {
			os.Exit(0)
		}
	}
}

func main() {
	initDB()
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("========================================")
	fmt.Println(" Select mode:")
	fmt.Println(" 1. Web UI")
	fmt.Println(" 2. CLI")
	fmt.Println("========================================")
	fmt.Print("Input (1 or 2): ")

	choice, _ := reader.ReadString('\n')
	if strings.TrimSpace(choice) == "1" {
		startWebServer()
	} else {
		startCLI()
	}
}
