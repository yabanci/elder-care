package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/arsenozhetov/elder-care/backend/internal/config"
	"github.com/arsenozhetov/elder-care/backend/internal/db"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	clear(ctx, pool)

	patient := mustUser(ctx, pool, "patient@demo.kz", "demo1234", "Айгүл Сейтова", "patient", "+77771112233", "1952-06-14", "ELDER001")
	doctor := mustUser(ctx, pool, "doctor@demo.kz", "demo1234", "Др. Нұрлан Ахметов", "doctor", "+77772223344", "", "")
	family := mustUser(ctx, pool, "family@demo.kz", "demo1234", "Ерлан Сейтов", "family", "+77773334455", "", "")

	mustLink(ctx, pool, patient, doctor, "doctor")
	mustLink(ctx, pool, patient, family, "family")

	seedMetrics(ctx, pool, patient)
	seedMedications(ctx, pool, patient)
	seedMessages(ctx, pool, patient, doctor, family)

	log.Println("seed complete")
	log.Println("  patient@demo.kz / demo1234  (invite code: ELDER001)")
	log.Println("  doctor@demo.kz  / demo1234")
	log.Println("  family@demo.kz  / demo1234")
}

func clear(ctx context.Context, pool *pgxpool.Pool) {
	// dev-only wipe to make seed idempotent
	_, err := pool.Exec(ctx, `
		TRUNCATE messages, medication_logs, medications, alerts, health_metrics, patient_links, users RESTART IDENTITY CASCADE
	`)
	if err != nil {
		log.Fatalf("truncate: %v", err)
	}
}

func mustUser(ctx context.Context, pool *pgxpool.Pool, email, pw, name, role, phone, birth, invite string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	var phoneArg, birthArg, inviteArg interface{}
	if phone != "" {
		phoneArg = phone
	}
	if birth != "" {
		t, _ := time.Parse("2006-01-02", birth)
		birthArg = t
	}
	if invite != "" {
		inviteArg = invite
	}
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO users(email, password_hash, full_name, role, phone, birth_date, invite_code)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id
	`, email, string(hash), name, role, phoneArg, birthArg, inviteArg).Scan(&id)
	if err != nil {
		log.Fatalf("insert user %s: %v", email, err)
	}
	return id
}

func mustLink(ctx context.Context, pool *pgxpool.Pool, patient, linked, rel string) {
	_, err := pool.Exec(ctx, `
		INSERT INTO patient_links(patient_id, linked_id, relation) VALUES($1,$2,$3)
	`, patient, linked, rel)
	if err != nil {
		log.Fatalf("link %s: %v", rel, err)
	}
}

func seedMetrics(ctx context.Context, pool *pgxpool.Pool, patient string) {
	rng := rand.New(rand.NewSource(42))
	now := time.Now()

	type profile struct {
		kind       string
		mean, std  float64
		outlierVal float64
	}
	profiles := []profile{
		{"pulse", 72, 3, 95},
		{"bp_sys", 130, 5, 168},
		{"bp_dia", 82, 4, 102},
		{"glucose", 5.6, 0.4, 9.8},
		{"temperature", 36.6, 0.2, 37.9},
		{"spo2", 97, 1, 92},
		{"weight", 68, 0.3, 0},
	}

	for _, p := range profiles {
		for i := 30; i >= 1; i-- {
			measuredAt := now.Add(-time.Duration(i) * 24 * time.Hour).Add(time.Duration(rng.Intn(12)) * time.Hour)
			v := p.mean + rng.NormFloat64()*p.std
			_, err := pool.Exec(ctx, `
				INSERT INTO health_metrics(patient_id, kind, value, measured_at)
				VALUES($1,$2,$3,$4)
			`, patient, p.kind, v, measuredAt)
			if err != nil {
				log.Fatalf("metric: %v", err)
			}
		}
		// inject one outlier 2 days ago for demo alerts
		if p.outlierVal != 0 {
			_, err := pool.Exec(ctx, `
				INSERT INTO health_metrics(patient_id, kind, value, measured_at, note)
				VALUES($1,$2,$3,$4,$5)
			`, patient, p.kind, p.outlierVal, now.Add(-2*24*time.Hour), "auto-seeded outlier")
			if err != nil {
				log.Fatalf("outlier: %v", err)
			}
		}
	}
	// run a lightweight baseline check on the last inserted outliers by inserting alert rows directly
	// (в продакшене alert создаёт API; для сидов пробежим простой trigger)
	rows, err := pool.Query(ctx, `
		WITH stats AS (
		  SELECT patient_id, kind, avg(value) AS mean, stddev_pop(value) AS std
		  FROM health_metrics WHERE patient_id=$1 GROUP BY patient_id, kind
		)
		SELECT hm.id, hm.patient_id, hm.kind, hm.value, s.mean, s.std
		FROM health_metrics hm
		JOIN stats s USING (patient_id, kind)
		WHERE hm.patient_id=$1 AND s.std > 0 AND abs(hm.value - s.mean)/s.std >= 2
	`, patient)
	if err != nil {
		log.Fatalf("baseline query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var metricID, pid, kind string
		var v, mean, std float64
		_ = rows.Scan(&metricID, &pid, &kind, &v, &mean, &std)
		z := abs((v - mean) / std)
		severity := "warning"
		reason := "отклонение от личной нормы (z≥2)"
		if z >= 3 {
			severity = "critical"
			reason = "значительное отклонение от личной нормы (z≥3)"
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO alerts(patient_id, metric_id, severity, reason, kind, value, baseline_mean, baseline_std)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		`, pid, metricID, severity, reason, kind, v, mean, std)
		if err != nil {
			log.Fatalf("alert: %v", err)
		}
	}
}

func seedMedications(ctx context.Context, pool *pgxpool.Pool, patient string) {
	meds := []struct {
		name   string
		dose   string
		times  []string
		notes  string
	}{
		{"Лизиноприл", "10 мг", []string{"08:00"}, "Артериальное давление"},
		{"Метформин", "500 мг", []string{"08:00", "20:00"}, "Диабет 2 типа"},
		{"Аспирин", "75 мг", []string{"08:00"}, "Профилактика"},
	}
	for _, m := range meds {
		_, err := pool.Exec(ctx, `
			INSERT INTO medications(patient_id, name, dosage, times_of_day, notes)
			VALUES($1,$2,$3,$4,$5)
		`, patient, m.name, m.dose, m.times, m.notes)
		if err != nil {
			log.Fatalf("med: %v", err)
		}
	}
}

func seedMessages(ctx context.Context, pool *pgxpool.Pool, patient, doctor, family string) {
	pairs := []struct {
		from, to string
		body     string
		ago      time.Duration
	}{
		{doctor, patient, "Добрый день, Айгүл. Как вы себя чувствуете сегодня?", 6 * time.Hour},
		{patient, doctor, "Здравствуйте, доктор. Давление немного повышено.", 5 * time.Hour},
		{doctor, patient, "Измерьте через час ещё раз и отпишитесь, пожалуйста.", 4 * time.Hour},
		{family, patient, "Мама, не забудь принять таблетки вечером!", 3 * time.Hour},
	}
	for _, p := range pairs {
		_, err := pool.Exec(ctx, `
			INSERT INTO messages(sender_id, recipient_id, body, created_at)
			VALUES($1,$2,$3, now() - $4::interval)
		`, p.from, p.to, p.body, p.ago.String())
		if err != nil {
			log.Fatalf("msg: %v", err)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
