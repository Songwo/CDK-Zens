package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cdk-airdrop-station/server/internal/model"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	sqlDialectMySQL    = "mysql"
	sqlDialectPostgres = "postgres"
)

// InitMySQL opens a connection pool and ensures all tables exist.
func InitMySQL(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	if err := autoMigrateMySQL(db); err != nil {
		return nil, fmt.Errorf("mysql migrate: %w", err)
	}
	return db, nil
}

// InitPostgres opens a PostgreSQL connection pool and ensures all tables exist.
func InitPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	if err := autoMigratePostgres(db); err != nil {
		return nil, fmt.Errorf("postgres migrate: %w", err)
	}
	return db, nil
}

func autoMigrateMySQL(db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(32) PRIMARY KEY, username VARCHAR(100) DEFAULT '', password VARCHAR(255) DEFAULT 'sso_community',
			role VARCHAR(20) DEFAULT 'user', community_user_id VARCHAR(100) DEFAULT '', avatar TEXT,
			nickname VARCHAR(100) DEFAULT '', email VARCHAR(200) DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uk_cuid (community_user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS projects (
			id VARCHAR(32) PRIMARY KEY, name VARCHAR(200) NOT NULL, description TEXT, status VARCHAR(20) DEFAULT 'active',
			creator_id VARCHAR(32) DEFAULT '', created_at VARCHAR(30) NOT NULL, updated_at VARCHAR(30) NOT NULL,
			INDEX idx_creator (creator_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS campaigns (
			id VARCHAR(32) PRIMARY KEY, project_id VARCHAR(32) NOT NULL, name VARCHAR(200) NOT NULL, description TEXT,
			status VARCHAR(20) DEFAULT 'active', total_stock INT DEFAULT 0, claimed_count INT DEFAULT 0, remaining_count INT DEFAULT 0,
			start_at VARCHAR(30) DEFAULT '', end_at VARCHAR(30) DEFAULT '', allow_repeat TINYINT(1) DEFAULT 0,
			per_user_limit INT DEFAULT 1, per_ip_limit INT DEFAULT 0, per_device_limit INT DEFAULT 0,
			require_captcha_default TINYINT(1) DEFAULT 1, project_code VARCHAR(20) DEFAULT '', enabled TINYINT(1) DEFAULT 1,
			rules TEXT, created_at VARCHAR(30) NOT NULL, updated_at VARCHAR(30) NOT NULL,
			INDEX idx_project (project_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS cdks (
			id VARCHAR(32) PRIMARY KEY, campaign_id VARCHAR(32) NOT NULL, code VARCHAR(500) NOT NULL,
			status VARCHAR(20) DEFAULT 'unused', claimed_by_record_id VARCHAR(32) DEFAULT '', claimed_at VARCHAR(30) DEFAULT '',
			node_id VARCHAR(32) DEFAULT '', created_at VARCHAR(30) NOT NULL, updated_at VARCHAR(30) NOT NULL,
			INDEX idx_cs (campaign_id, status), INDEX idx_code (code(100))
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS nodes (
			id VARCHAR(32) PRIMARY KEY, project_id VARCHAR(32) DEFAULT '', campaign_id VARCHAR(32) NOT NULL,
			name VARCHAR(200) NOT NULL, slug VARCHAR(100) NOT NULL, status VARCHAR(20) DEFAULT 'active',
			title VARCHAR(200) DEFAULT '', description TEXT, button_text VARCHAR(100) DEFAULT '立即领取',
			require_captcha TINYINT(1) DEFAULT 1, show_stock TINYINT(1) DEFAULT 1, show_end_time TINYINT(1) DEFAULT 1,
			visits INT DEFAULT 0, unique_visitors INT DEFAULT 0, claims INT DEFAULT 0, failed_claims INT DEFAULT 0,
			last_visited_at VARCHAR(30) DEFAULT '', limit_val INT DEFAULT 0, ip_limit_enabled TINYINT(1) DEFAULT 0,
			device_limit_enabled TINYINT(1) DEFAULT 0, created_at VARCHAR(30) NOT NULL, updated_at VARCHAR(30) NOT NULL,
			UNIQUE KEY uk_slug (slug), INDEX idx_project (project_id), INDEX idx_campaign (campaign_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS claim_records (
			id VARCHAR(32) PRIMARY KEY, campaign_id VARCHAR(32) DEFAULT '', node_id VARCHAR(32) DEFAULT '',
			project_id VARCHAR(32) DEFAULT '', cdk_id VARCHAR(32) DEFAULT '', code VARCHAR(500) DEFAULT '',
			status VARCHAR(20) NOT NULL, reason TEXT, ip VARCHAR(50) DEFAULT '', user_agent TEXT,
			fingerprint VARCHAR(200) DEFAULT '', idempotency_key VARCHAR(200) DEFAULT '', claim_token VARCHAR(100) DEFAULT '',
			hcaptcha_passed TINYINT(1) DEFAULT 0, risk_hit TINYINT(1) DEFAULT 0, risk_rule_ids TEXT,
			user_id VARCHAR(32) DEFAULT '', reward_content TEXT, created_at VARCHAR(30) NOT NULL,
			INDEX idx_campaign (campaign_id), INDEX idx_node (node_id), INDEX idx_project (project_id),
			INDEX idx_user (user_id), INDEX idx_fp (fingerprint), INDEX idx_ip (ip), INDEX idx_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS risk_rules (
			id VARCHAR(32) PRIMARY KEY, name VARCHAR(200) NOT NULL, type VARCHAR(50) NOT NULL,
			enabled TINYINT(1) DEFAULT 1, config JSON, action VARCHAR(50) DEFAULT 'block',
			created_at VARCHAR(30) NOT NULL, updated_at VARCHAR(30) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS blacklist (
			id VARCHAR(32) PRIMARY KEY, type VARCHAR(50) NOT NULL, value VARCHAR(500) NOT NULL,
			reason TEXT, enabled TINYINT(1) DEFAULT 1, created_at VARCHAR(30) NOT NULL, updated_at VARCHAR(30) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS system_logs (
			id VARCHAR(32) PRIMARY KEY, type VARCHAR(30) DEFAULT 'operation', level VARCHAR(10) DEFAULT 'info',
			title VARCHAR(200) DEFAULT '', message TEXT, actor VARCHAR(100) DEFAULT '', ip VARCHAR(50) DEFAULT '',
			target_type VARCHAR(50) DEFAULT '', target_id VARCHAR(50) DEFAULT '', metadata JSON, created_at VARCHAR(30) NOT NULL,
			INDEX idx_type (type), INDEX idx_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS export_tasks (
			id VARCHAR(32) PRIMARY KEY, type VARCHAR(50) NOT NULL, status VARCHAR(20) DEFAULT 'pending',
			filename VARCHAR(200) DEFAULT '', file_path VARCHAR(500) DEFAULT '', filter JSON,
			error_msg TEXT, created_at VARCHAR(30) NOT NULL, finished_at VARCHAR(30) DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS settings (
			id INT PRIMARY KEY DEFAULT 1, system_name VARCHAR(100) DEFAULT 'Zens-CDK',
			brand_name VARCHAR(100) DEFAULT 'Zens-CDK', brand_english_name VARCHAR(100) DEFAULT 'Zens CDK Airdrop Hub',
			logo_text VARCHAR(20) DEFAULT 'ZC', public_base_url VARCHAR(500) DEFAULT '', storage_mode VARCHAR(20) DEFAULT 'mysql',
			redis_enabled TINYINT(1) DEFAULT 0, rabbitmq_enabled TINYINT(1) DEFAULT 0,
			created_at VARCHAR(30) NOT NULL, updated_at VARCHAR(30) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS captcha_config (
			id INT PRIMARY KEY DEFAULT 1, provider VARCHAR(20) DEFAULT 'hcaptcha', enabled TINYINT(1) DEFAULT 1,
			last_test_status VARCHAR(20) DEFAULT '', last_test_message TEXT, last_test_at VARCHAR(30) DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, ddl := range tables {
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("exec DDL: %w\nSQL: %s", err, sqlSnippet(ddl))
		}
	}
	// seed defaults
	now := nowISO()
	db.Exec("INSERT IGNORE INTO settings (id,created_at,updated_at) VALUES (1,?,?)", now, now)
	db.Exec("INSERT IGNORE INTO captcha_config (id) VALUES (1)")
	return nil
}

func autoMigratePostgres(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(32) PRIMARY KEY,
			username VARCHAR(100) NOT NULL DEFAULT '',
			password VARCHAR(255) NOT NULL DEFAULT 'sso_community',
			role VARCHAR(20) NOT NULL DEFAULT 'user',
			community_user_id VARCHAR(100) DEFAULT '',
			avatar TEXT,
			nickname VARCHAR(100) DEFAULT '',
			email VARCHAR(200) DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_users_community_user_id ON users (community_user_id) WHERE community_user_id <> ''`,
		`CREATE TABLE IF NOT EXISTS projects (
			id VARCHAR(32) PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			description TEXT,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			creator_id VARCHAR(32) DEFAULT '',
			created_at VARCHAR(30) NOT NULL,
			updated_at VARCHAR(30) NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_creator ON projects (creator_id)`,
		`CREATE TABLE IF NOT EXISTS campaigns (
			id VARCHAR(32) PRIMARY KEY,
			project_id VARCHAR(32) NOT NULL,
			name VARCHAR(200) NOT NULL,
			description TEXT,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			total_stock INT NOT NULL DEFAULT 0,
			claimed_count INT NOT NULL DEFAULT 0,
			remaining_count INT NOT NULL DEFAULT 0,
			start_at VARCHAR(30) DEFAULT '',
			end_at VARCHAR(30) DEFAULT '',
			allow_repeat BOOLEAN NOT NULL DEFAULT FALSE,
			per_user_limit INT NOT NULL DEFAULT 1,
			per_ip_limit INT NOT NULL DEFAULT 0,
			per_device_limit INT NOT NULL DEFAULT 0,
			require_captcha_default BOOLEAN NOT NULL DEFAULT TRUE,
			project_code VARCHAR(20) DEFAULT '',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			rules TEXT,
			created_at VARCHAR(30) NOT NULL,
			updated_at VARCHAR(30) NOT NULL
		)`,
		`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS project_id VARCHAR(32) DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_project ON campaigns (project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns (status)`,
		`CREATE TABLE IF NOT EXISTS cdks (
			id VARCHAR(32) PRIMARY KEY,
			campaign_id VARCHAR(32) NOT NULL,
			code VARCHAR(500) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'unused',
			claimed_by_record_id VARCHAR(32) DEFAULT '',
			claimed_at VARCHAR(30) DEFAULT '',
			node_id VARCHAR(32) DEFAULT '',
			created_at VARCHAR(30) NOT NULL,
			updated_at VARCHAR(30) NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cdks_campaign_status ON cdks (campaign_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_cdks_code ON cdks (code)`,
		`CREATE TABLE IF NOT EXISTS nodes (
			id VARCHAR(32) PRIMARY KEY,
			project_id VARCHAR(32) NOT NULL DEFAULT '',
			campaign_id VARCHAR(32) NOT NULL,
			name VARCHAR(200) NOT NULL,
			slug VARCHAR(100) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			title VARCHAR(200) DEFAULT '',
			description TEXT,
			button_text VARCHAR(100) DEFAULT '立即领取',
			require_captcha BOOLEAN NOT NULL DEFAULT TRUE,
			show_stock BOOLEAN NOT NULL DEFAULT TRUE,
			show_end_time BOOLEAN NOT NULL DEFAULT TRUE,
			visits INT NOT NULL DEFAULT 0,
			unique_visitors INT NOT NULL DEFAULT 0,
			claims INT NOT NULL DEFAULT 0,
			failed_claims INT NOT NULL DEFAULT 0,
			last_visited_at VARCHAR(30) DEFAULT '',
			limit_val INT NOT NULL DEFAULT 0,
			ip_limit_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			device_limit_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			created_at VARCHAR(30) NOT NULL,
			updated_at VARCHAR(30) NOT NULL
		)`,
		`ALTER TABLE nodes ADD COLUMN IF NOT EXISTS project_id VARCHAR(32) DEFAULT ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_nodes_slug ON nodes (slug)`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_project ON nodes (project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_campaign ON nodes (campaign_id)`,
		`CREATE TABLE IF NOT EXISTS claim_records (
			id VARCHAR(32) PRIMARY KEY,
			campaign_id VARCHAR(32) NOT NULL DEFAULT '',
			node_id VARCHAR(32) NOT NULL DEFAULT '',
			project_id VARCHAR(32) NOT NULL DEFAULT '',
			cdk_id VARCHAR(32) DEFAULT '',
			code VARCHAR(500) DEFAULT '',
			status VARCHAR(20) NOT NULL,
			reason TEXT,
			ip VARCHAR(50) DEFAULT '',
			user_agent TEXT,
			fingerprint VARCHAR(200) DEFAULT '',
			idempotency_key VARCHAR(200) DEFAULT '',
			claim_token VARCHAR(100) DEFAULT '',
			hcaptcha_passed BOOLEAN NOT NULL DEFAULT FALSE,
			risk_hit BOOLEAN NOT NULL DEFAULT FALSE,
			risk_rule_ids TEXT,
			user_id VARCHAR(32) DEFAULT '',
			reward_content TEXT,
			created_at VARCHAR(30) NOT NULL
		)`,
		`ALTER TABLE claim_records ADD COLUMN IF NOT EXISTS project_id VARCHAR(32) DEFAULT ''`,
		`ALTER TABLE claim_records ADD COLUMN IF NOT EXISTS user_id VARCHAR(32) DEFAULT ''`,
		`ALTER TABLE claim_records ADD COLUMN IF NOT EXISTS reward_content TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_claim_records_campaign ON claim_records (campaign_id)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_records_node ON claim_records (node_id)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_records_project ON claim_records (project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_records_fingerprint ON claim_records (fingerprint)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_records_user ON claim_records (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_records_ip ON claim_records (ip)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_records_created ON claim_records (created_at)`,
		`CREATE TABLE IF NOT EXISTS risk_rules (
			id VARCHAR(32) PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			type VARCHAR(50) NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			config TEXT,
			action VARCHAR(50) DEFAULT 'block',
			created_at VARCHAR(30) NOT NULL,
			updated_at VARCHAR(30) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS blacklist (
			id VARCHAR(32) PRIMARY KEY,
			type VARCHAR(50) NOT NULL,
			value VARCHAR(500) NOT NULL,
			reason TEXT,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at VARCHAR(30) NOT NULL,
			updated_at VARCHAR(30) NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_blacklist_type_value ON blacklist (type, value)`,
		`CREATE TABLE IF NOT EXISTS system_logs (
			id VARCHAR(32) PRIMARY KEY,
			type VARCHAR(30) NOT NULL DEFAULT 'operation',
			level VARCHAR(10) NOT NULL DEFAULT 'info',
			title VARCHAR(200) DEFAULT '',
			message TEXT,
			actor VARCHAR(100) DEFAULT '',
			ip VARCHAR(50) DEFAULT '',
			target_type VARCHAR(50) DEFAULT '',
			target_id VARCHAR(50) DEFAULT '',
			metadata TEXT,
			created_at VARCHAR(30) NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_type ON system_logs (type)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_created ON system_logs (created_at)`,
		`CREATE TABLE IF NOT EXISTS export_tasks (
			id VARCHAR(32) PRIMARY KEY,
			type VARCHAR(50) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			filename VARCHAR(200) DEFAULT '',
			file_path VARCHAR(500) DEFAULT '',
			filter TEXT,
			error_msg TEXT,
			created_at VARCHAR(30) NOT NULL,
			finished_at VARCHAR(30) DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			id INT PRIMARY KEY DEFAULT 1,
			system_name VARCHAR(100) DEFAULT 'Zens-CDK',
			brand_name VARCHAR(100) DEFAULT 'Zens-CDK',
			brand_english_name VARCHAR(100) DEFAULT 'Zens CDK Airdrop Hub',
			logo_text VARCHAR(20) DEFAULT 'ZC',
			public_base_url VARCHAR(500) DEFAULT '',
			storage_mode VARCHAR(20) DEFAULT 'postgres',
			redis_enabled BOOLEAN DEFAULT FALSE,
			rabbitmq_enabled BOOLEAN DEFAULT FALSE,
			created_at VARCHAR(30) NOT NULL,
			updated_at VARCHAR(30) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS captcha_config (
			id INT PRIMARY KEY DEFAULT 1,
			provider VARCHAR(20) DEFAULT 'hcaptcha',
			enabled BOOLEAN DEFAULT TRUE,
			last_test_status VARCHAR(20) DEFAULT '',
			last_test_message TEXT,
			last_test_at VARCHAR(30) DEFAULT ''
		)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec DDL: %w\nSQL: %s", err, sqlSnippet(stmt))
		}
	}
	now := nowISO()
	db.Exec("INSERT INTO settings (id,created_at,updated_at) VALUES (1,$1,$2) ON CONFLICT (id) DO NOTHING", now, now)
	db.Exec("INSERT INTO captcha_config (id) VALUES (1) ON CONFLICT (id) DO NOTHING")
	return nil
}

// ── Load all data from SQL into memory ───────────────────────────

func (s *Store) loadFromSQL() error {
	db := s.db
	if db == nil {
		return nil
	}
	// Users
	rows, _ := db.Query("SELECT id,username,password,role,community_user_id,COALESCE(avatar,''),nickname,email,created_at,updated_at FROM users")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var u model.User
			var ca time.Time
			var ua time.Time
			rows.Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.CommunityUserID, &u.Avatar, &u.Nickname, &u.Email, &ca, &ua)
			u.CreatedAt = ca
			u.UpdatedAt = ua.Format(timeLayout)
			s.data.Users[u.ID] = &u
			s.data.Admins[u.ID] = &u
		}
	}
	// Projects
	rows2, _ := db.Query("SELECT id,name,COALESCE(description,''),status,COALESCE(creator_id,''),created_at,updated_at FROM projects")
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var p model.Project
			rows2.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CreatorID, &p.CreatedAt, &p.UpdatedAt)
			p.CampaignIDs = []string{}
			p.NodeIDs = []string{}
			s.data.Projects[p.ID] = &p
		}
	}
	// Campaigns
	rows3, _ := db.Query("SELECT id,project_id,name,COALESCE(description,''),status,total_stock,claimed_count,remaining_count,COALESCE(start_at,''),COALESCE(end_at,''),allow_repeat,per_user_limit,per_ip_limit,per_device_limit,require_captcha_default,COALESCE(project_code,''),enabled,COALESCE(rules,''),created_at,updated_at FROM campaigns")
	if rows3 != nil {
		defer rows3.Close()
		for rows3.Next() {
			var c model.Campaign
			rows3.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Description, &c.Status, &c.TotalStock, &c.ClaimedCount, &c.RemainingCount,
				&c.StartAt, &c.EndAt, &c.AllowRepeat, &c.PerUserLimit, &c.PerIPLimit, &c.PerDeviceLimit,
				&c.RequireCaptchaDefault, &c.ProjectCode, &c.Enabled, &c.Rules, &c.CreatedAt, &c.UpdatedAt)
			c.StartTime = c.StartAt
			c.EndTime = c.EndAt
			c.NodeIDs = []string{}
			s.data.Campaigns[c.ID] = &c
			if p := s.data.Projects[c.ProjectID]; p != nil {
				p.CampaignIDs = appendUnique(p.CampaignIDs, c.ID)
			}
		}
	}
	// CDKs
	rows4, _ := db.Query("SELECT id,campaign_id,code,status,COALESCE(claimed_by_record_id,''),COALESCE(claimed_at,''),COALESCE(node_id,''),created_at,updated_at FROM cdks")
	if rows4 != nil {
		defer rows4.Close()
		for rows4.Next() {
			var k model.CDK
			rows4.Scan(&k.ID, &k.CampaignID, &k.Code, &k.Status, &k.ClaimedByRecordID, &k.ClaimedAt, &k.NodeID, &k.CreatedAt, &k.UpdatedAt)
			s.data.CDKs[k.ID] = &k
		}
	}
	// Nodes
	rows5, _ := db.Query("SELECT id,COALESCE(project_id,''),campaign_id,name,slug,status,COALESCE(title,''),COALESCE(description,''),COALESCE(button_text,'立即领取'),require_captcha,show_stock,show_end_time,visits,unique_visitors,claims,failed_claims,COALESCE(last_visited_at,''),limit_val,ip_limit_enabled,device_limit_enabled,created_at,updated_at FROM nodes")
	if rows5 != nil {
		defer rows5.Close()
		for rows5.Next() {
			var n model.DistributionNode
			rows5.Scan(&n.ID, &n.ProjectID, &n.CampaignID, &n.Name, &n.Slug, &n.Status, &n.Title, &n.Description,
				&n.ButtonText, &n.RequireCaptcha, &n.ShowStock, &n.ShowEndTime, &n.Visits, &n.UniqueVisitors,
				&n.Claims, &n.FailedClaims, &n.LastVisitedAt, &n.Limit, &n.IPLimitEnabled, &n.DeviceLimitEnabled,
				&n.CreatedAt, &n.UpdatedAt)
			s.data.Nodes[n.ID] = &n
			if c := s.data.Campaigns[n.CampaignID]; c != nil {
				c.NodeIDs = appendUnique(c.NodeIDs, n.ID)
			}
			if p := s.data.Projects[n.ProjectID]; p != nil {
				p.NodeIDs = appendUnique(p.NodeIDs, n.ID)
			}
		}
	}
	// ClaimRecords
	rows6, _ := db.Query("SELECT id,COALESCE(campaign_id,''),COALESCE(node_id,''),COALESCE(project_id,''),COALESCE(cdk_id,''),COALESCE(code,''),status,COALESCE(reason,''),COALESCE(ip,''),COALESCE(user_agent,''),COALESCE(fingerprint,''),COALESCE(idempotency_key,''),COALESCE(claim_token,''),hcaptcha_passed,risk_hit,COALESCE(risk_rule_ids,''),COALESCE(user_id,''),COALESCE(reward_content,''),created_at FROM claim_records")
	if rows6 != nil {
		defer rows6.Close()
		for rows6.Next() {
			var r model.ClaimRecord
			var ruleIDs string
			rows6.Scan(&r.ID, &r.CampaignID, &r.NodeID, &r.ProjectID, &r.CDKID, &r.Code, &r.Status,
				&r.Reason, &r.IP, &r.UserAgent, &r.Fingerprint, &r.IdempotencyKey, &r.ClaimToken,
				&r.HCaptchaPassed, &r.RiskHit, &ruleIDs, &r.UserID, &r.RewardContent, &r.CreatedAt)
			if ruleIDs != "" {
				r.RiskRuleIDs = strings.Split(ruleIDs, ",")
			}
			s.data.ClaimRecords[r.ID] = &r
		}
	}
	// RiskRules
	rows7, _ := db.Query("SELECT id,name,type,enabled,COALESCE(config,'{}'),COALESCE(action,'block'),created_at,updated_at FROM risk_rules")
	if rows7 != nil {
		defer rows7.Close()
		for rows7.Next() {
			var r model.RiskRule
			var configJSON string
			rows7.Scan(&r.ID, &r.Name, &r.Type, &r.Enabled, &configJSON, &r.Action, &r.CreatedAt, &r.UpdatedAt)
			json.Unmarshal([]byte(configJSON), &r.Config)
			s.data.RiskRules[r.ID] = &r
		}
	}
	// Blacklist
	rows8, _ := db.Query("SELECT id,type,value,COALESCE(reason,''),enabled,created_at,updated_at FROM blacklist")
	if rows8 != nil {
		defer rows8.Close()
		for rows8.Next() {
			var b model.BlacklistItem
			rows8.Scan(&b.ID, &b.Type, &b.Value, &b.Reason, &b.Enabled, &b.CreatedAt, &b.UpdatedAt)
			s.data.Blacklist[b.ID] = &b
		}
	}
	// SystemLogs
	rows9, _ := db.Query("SELECT id,type,level,COALESCE(title,''),COALESCE(message,''),COALESCE(actor,''),COALESCE(ip,''),COALESCE(target_type,''),COALESCE(target_id,''),COALESCE(metadata,'{}'),created_at FROM system_logs ORDER BY created_at DESC LIMIT 5000")
	if rows9 != nil {
		defer rows9.Close()
		for rows9.Next() {
			var l model.SystemLog
			var metaJSON string
			rows9.Scan(&l.ID, &l.Type, &l.Level, &l.Title, &l.Message, &l.Actor, &l.IP, &l.TargetType, &l.TargetID, &metaJSON, &l.CreatedAt)
			json.Unmarshal([]byte(metaJSON), &l.Metadata)
			s.data.SystemLogs[l.ID] = &l
		}
	}
	// Settings
	var st model.Settings
	row := db.QueryRow("SELECT system_name,brand_name,brand_english_name,logo_text,COALESCE(public_base_url,''),storage_mode,redis_enabled,rabbitmq_enabled,created_at,updated_at FROM settings WHERE id=1")
	if err := row.Scan(&st.SystemName, &st.BrandName, &st.BrandEnglishName, &st.LogoText, &st.PublicBaseURL, &st.StorageMode, &st.RedisEnabled, &st.RabbitMQEnabled, &st.CreatedAt, &st.UpdatedAt); err == nil {
		s.data.Settings = st
	}
	// CaptchaConfig
	var cc model.CaptchaConfig
	row2 := db.QueryRow("SELECT provider,enabled,COALESCE(last_test_status,''),COALESCE(last_test_message,''),COALESCE(last_test_at,'') FROM captcha_config WHERE id=1")
	if err := row2.Scan(&cc.Provider, &cc.Enabled, &cc.LastTestStatus, &cc.LastTestMessage, &cc.LastTestAt); err == nil {
		s.data.CaptchaConfig = cc
	}
	// ExportTasks
	rows10, _ := db.Query("SELECT id,type,status,COALESCE(filename,''),COALESCE(file_path,''),COALESCE(filter,'{}'),COALESCE(error_msg,''),created_at,COALESCE(finished_at,'') FROM export_tasks")
	if rows10 != nil {
		defer rows10.Close()
		for rows10.Next() {
			var t model.ExportTask
			var filterJSON string
			rows10.Scan(&t.ID, &t.Type, &t.Status, &t.Filename, &t.FilePath, &filterJSON, &t.Error, &t.CreatedAt, &t.FinishedAt)
			json.Unmarshal([]byte(filterJSON), &t.Filter)
			s.data.ExportTasks[t.ID] = &t
		}
	}
	return nil
}

// ── Per-entity SQL save helpers ─────────────────────────────────

func (s *Store) dbSaveUser(u *model.User) {
	columns := []string{"id", "username", "password", "role", "community_user_id", "avatar", "nickname", "email", "created_at", "updated_at"}
	s.dbUpsert("users", columns, "id",
		u.ID, u.Username, u.Password, u.Role, u.CommunityUserID, u.Avatar, u.Nickname, u.Email, normalizedUserCreatedAt(u), normalizedUserUpdatedAt(u))
}

func (s *Store) dbSaveProject(p *model.Project) {
	columns := []string{"id", "name", "description", "status", "creator_id", "created_at", "updated_at"}
	s.dbUpsert("projects", columns, "id", p.ID, p.Name, p.Description, p.Status, p.CreatorID, p.CreatedAt, p.UpdatedAt)
}

func (s *Store) dbDeleteProject(id string) {
	s.dbExec(fmt.Sprintf("DELETE FROM projects WHERE id=%s", s.dbPlaceholder(1)), id)
}

func (s *Store) dbSaveCampaign(c *model.Campaign) {
	columns := []string{"id", "project_id", "name", "description", "status", "total_stock", "claimed_count", "remaining_count", "start_at", "end_at", "allow_repeat", "per_user_limit", "per_ip_limit", "per_device_limit", "require_captcha_default", "project_code", "enabled", "rules", "created_at", "updated_at"}
	s.dbUpsert("campaigns", columns, "id",
		c.ID, c.ProjectID, c.Name, c.Description, c.Status, c.TotalStock, c.ClaimedCount, c.RemainingCount,
		c.StartAt, c.EndAt, c.AllowRepeat, c.PerUserLimit, c.PerIPLimit, c.PerDeviceLimit,
		c.RequireCaptchaDefault, c.ProjectCode, c.Enabled, c.Rules, c.CreatedAt, c.UpdatedAt)
}

func (s *Store) dbDeleteCampaign(id string) {
	s.dbExec(fmt.Sprintf("DELETE FROM campaigns WHERE id=%s", s.dbPlaceholder(1)), id)
	s.dbExec(fmt.Sprintf("DELETE FROM cdks WHERE campaign_id=%s", s.dbPlaceholder(1)), id)
}

func (s *Store) dbSaveCDK(k *model.CDK) {
	columns := []string{"id", "campaign_id", "code", "status", "claimed_by_record_id", "claimed_at", "node_id", "created_at", "updated_at"}
	s.dbUpsert("cdks", columns, "id", k.ID, k.CampaignID, k.Code, k.Status, k.ClaimedByRecordID, k.ClaimedAt, k.NodeID, k.CreatedAt, k.UpdatedAt)
}

func (s *Store) dbDeleteCDK(id string) {
	s.dbExec(fmt.Sprintf("DELETE FROM cdks WHERE id=%s", s.dbPlaceholder(1)), id)
}

func (s *Store) dbSaveNode(n *model.DistributionNode) {
	columns := []string{"id", "project_id", "campaign_id", "name", "slug", "status", "title", "description", "button_text", "require_captcha", "show_stock", "show_end_time", "visits", "unique_visitors", "claims", "failed_claims", "last_visited_at", "limit_val", "ip_limit_enabled", "device_limit_enabled", "created_at", "updated_at"}
	s.dbUpsert("nodes", columns, "id",
		n.ID, n.ProjectID, n.CampaignID, n.Name, n.Slug, n.Status, n.Title, n.Description, n.ButtonText,
		n.RequireCaptcha, n.ShowStock, n.ShowEndTime, n.Visits, n.UniqueVisitors, n.Claims, n.FailedClaims,
		n.LastVisitedAt, n.Limit, n.IPLimitEnabled, n.DeviceLimitEnabled, n.CreatedAt, n.UpdatedAt)
}

func (s *Store) dbDeleteNode(id string) {
	s.dbExec(fmt.Sprintf("DELETE FROM nodes WHERE id=%s", s.dbPlaceholder(1)), id)
}

func (s *Store) dbSaveClaimRecord(r *model.ClaimRecord) {
	ruleIDs := strings.Join(r.RiskRuleIDs, ",")
	columns := []string{"id", "campaign_id", "node_id", "project_id", "cdk_id", "code", "status", "reason", "ip", "user_agent", "fingerprint", "idempotency_key", "claim_token", "hcaptcha_passed", "risk_hit", "risk_rule_ids", "user_id", "reward_content", "created_at"}
	s.dbUpsert("claim_records", columns, "id",
		r.ID, r.CampaignID, r.NodeID, r.ProjectID, r.CDKID, r.Code, r.Status, r.Reason, r.IP, r.UserAgent,
		r.Fingerprint, r.IdempotencyKey, r.ClaimToken, r.HCaptchaPassed, r.RiskHit, ruleIDs, r.UserID, r.RewardContent, r.CreatedAt)
}

func (s *Store) dbSaveRiskRule(r *model.RiskRule) {
	configJSON, _ := json.Marshal(r.Config)
	columns := []string{"id", "name", "type", "enabled", "config", "action", "created_at", "updated_at"}
	s.dbUpsert("risk_rules", columns, "id", r.ID, r.Name, r.Type, r.Enabled, string(configJSON), r.Action, r.CreatedAt, r.UpdatedAt)
}

func (s *Store) dbDeleteRiskRule(id string) {
	s.dbExec(fmt.Sprintf("DELETE FROM risk_rules WHERE id=%s", s.dbPlaceholder(1)), id)
}

func (s *Store) dbSaveBlacklist(b *model.BlacklistItem) {
	columns := []string{"id", "type", "value", "reason", "enabled", "created_at", "updated_at"}
	s.dbUpsert("blacklist", columns, "id", b.ID, b.Type, b.Value, b.Reason, b.Enabled, b.CreatedAt, b.UpdatedAt)
}

func (s *Store) dbDeleteBlacklist(id string) {
	s.dbExec(fmt.Sprintf("DELETE FROM blacklist WHERE id=%s", s.dbPlaceholder(1)), id)
}

func (s *Store) dbSaveLog(l *model.SystemLog) {
	metaJSON, _ := json.Marshal(l.Metadata)
	columns := []string{"id", "type", "level", "title", "message", "actor", "ip", "target_type", "target_id", "metadata", "created_at"}
	s.dbInsert("system_logs", columns, l.ID, l.Type, l.Level, l.Title, l.Message, l.Actor, l.IP, l.TargetType, l.TargetID, string(metaJSON), l.CreatedAt)
}

func (s *Store) dbDeleteOldLogs(before string) {
	s.dbExec(fmt.Sprintf("DELETE FROM system_logs WHERE created_at < %s", s.dbPlaceholder(1)), before)
}

func (s *Store) dbSaveSettings(st model.Settings) {
	columns := []string{"id", "system_name", "brand_name", "brand_english_name", "logo_text", "public_base_url", "storage_mode", "redis_enabled", "rabbitmq_enabled", "created_at", "updated_at"}
	s.dbUpsert("settings", columns, "id", 1, st.SystemName, st.BrandName, st.BrandEnglishName, st.LogoText, st.PublicBaseURL, st.StorageMode, st.RedisEnabled, st.RabbitMQEnabled, st.CreatedAt, st.UpdatedAt)
}

func (s *Store) dbSaveCaptchaConfig(cc model.CaptchaConfig) {
	columns := []string{"id", "provider", "enabled", "last_test_status", "last_test_message", "last_test_at"}
	s.dbUpsert("captcha_config", columns, "id", 1, cc.Provider, cc.Enabled, cc.LastTestStatus, cc.LastTestMessage, cc.LastTestAt)
}

func (s *Store) dbSaveExportTask(t *model.ExportTask) {
	filterJSON, _ := json.Marshal(t.Filter)
	columns := []string{"id", "type", "status", "filename", "file_path", "filter", "error_msg", "created_at", "finished_at"}
	s.dbUpsert("export_tasks", columns, "id", t.ID, t.Type, t.Status, t.Filename, t.FilePath, string(filterJSON), t.Error, t.CreatedAt, t.FinishedAt)
}

func (s *Store) dbUpsert(table string, columns []string, conflictColumn string, args ...interface{}) {
	if s.db == nil {
		return
	}
	query := mysqlReplaceSQL(table, columns)
	if s.dbDialect == sqlDialectPostgres {
		query = postgresUpsertSQL(table, columns, conflictColumn)
	}
	_, _ = s.db.Exec(query, args...)
}

func (s *Store) dbInsert(table string, columns []string, args ...interface{}) {
	if s.db == nil {
		return
	}
	query := mysqlInsertSQL(table, columns)
	if s.dbDialect == sqlDialectPostgres {
		query = postgresInsertSQL(table, columns)
	}
	_, _ = s.db.Exec(query, args...)
}

func (s *Store) dbExec(query string, args ...interface{}) {
	if s.db == nil {
		return
	}
	_, _ = s.db.Exec(query, args...)
}

func (s *Store) dbPlaceholder(index int) string {
	if s.dbDialect == sqlDialectPostgres {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func mysqlReplaceSQL(table string, columns []string) string {
	return fmt.Sprintf("REPLACE INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ","), strings.Join(repeat("?", len(columns)), ","))
}

func mysqlInsertSQL(table string, columns []string) string {
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ","), strings.Join(repeat("?", len(columns)), ","))
}

func postgresInsertSQL(table string, columns []string) string {
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quotePostgresIdent(table), joinPostgresIdents(columns), strings.Join(postgresPlaceholders(len(columns)), ","))
}

func postgresUpsertSQL(table string, columns []string, conflictColumn string) string {
	updates := make([]string, 0, len(columns)-1)
	for _, column := range columns {
		if column == conflictColumn {
			continue
		}
		quoted := quotePostgresIdent(column)
		updates = append(updates, fmt.Sprintf("%s=EXCLUDED.%s", quoted, quoted))
	}
	if len(updates) == 0 {
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO NOTHING",
			quotePostgresIdent(table), joinPostgresIdents(columns), strings.Join(postgresPlaceholders(len(columns)), ","), quotePostgresIdent(conflictColumn))
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		quotePostgresIdent(table), joinPostgresIdents(columns), strings.Join(postgresPlaceholders(len(columns)), ","), quotePostgresIdent(conflictColumn), strings.Join(updates, ","))
}

func postgresPlaceholders(count int) []string {
	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	return placeholders
}

func joinPostgresIdents(columns []string) string {
	quoted := make([]string, len(columns))
	for i, column := range columns {
		quoted[i] = quotePostgresIdent(column)
	}
	return strings.Join(quoted, ",")
}

func quotePostgresIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func repeat(value string, count int) []string {
	items := make([]string, count)
	for i := range items {
		items[i] = value
	}
	return items
}

func normalizedUserCreatedAt(u *model.User) time.Time {
	if u.CreatedAt.IsZero() {
		return time.Now()
	}
	return u.CreatedAt
}

func normalizedUserUpdatedAt(u *model.User) time.Time {
	if value := strings.TrimSpace(u.UpdatedAt); value != "" {
		if parsed, err := time.Parse(timeLayout, value); err == nil {
			return parsed
		}
	}
	return normalizedUserCreatedAt(u)
}

func sqlSnippet(sql string) string {
	sql = strings.TrimSpace(sql)
	if len(sql) <= 80 {
		return sql
	}
	return sql[:80]
}
