package migrate

import (
	"ferry/database"
	"ferry/global/orm"
	"ferry/models/gorm"
	"ferry/models/system"
	"ferry/pkg/logger"
	config2 "ferry/tools/config"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	config   string
	mode     string
	StartCmd = &cobra.Command{
		Use:   "init",
		Short: "initialize the database",
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&config, "config", "c", "config/settings.yml", "Start server with provided configuration file")
	StartCmd.PersistentFlags().StringVarP(&mode, "mode", "m", "dev", "server mode ; eg:dev,test,prod")
}

func run() {
	usage := `start init`
	fmt.Println(usage)
	//1. 读取配置
	config2.ConfigSetup(config)
	//2. 初始化数据库链接
	database.Setup()
	//3. 数据库迁移
	_ = migrateModel()
	logger.Info("数据库结构初始化成功！")
	//4. 数据初始化完成
	if err := system.InitDb(); err != nil {
		logger.Fatalf("数据库基础数据初始化失败，%v", err)
	}

	usage = `数据库基础数据初始化成功`
	fmt.Println(usage)
}

func migrateModel() error {
	if config2.DatabaseConfig.Dbtype == "mysql" {
		orm.Eloquent = orm.Eloquent.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4")
	}
	err := gorm.AutoMigrate(orm.Eloquent)
	if err != nil {
		return err
	}
	// 创建 sys_notify_user_mapping 表（无 Go model，需手动建表）
	return orm.Eloquent.Exec(
		`CREATE TABLE IF NOT EXISTS sys_notify_user_mapping (
			id int(11) NOT NULL AUTO_INCREMENT,
			user_id int(11) NOT NULL COMMENT '系统用户ID',
			channel int(11) NOT NULL COMMENT '渠道类型',
			open_id varchar(128) DEFAULT NULL COMMENT '第三方用户标识',
			PRIMARY KEY (id),
			KEY idx_user_id (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	).Error
}
