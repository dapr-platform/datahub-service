package database

import (
	"datahub-service/service/database/views"
	"fmt"

	"gorm.io/gorm"
)

func AutoMigrateView(db *gorm.DB) error {
	allViews := make(map[string]map[string]string)
	allViews["basic_library"] = views.BasicLibraryViews
	allViews["thematic_library"] = views.ThematicLibraryViews
	allViews["sync_tasks"] = views.SyncTasksViews
	for _, viewSQLs := range allViews {
		for name, viewSQL := range viewSQLs {
			if err := db.Exec(viewSQL).Error; err != nil {
				return fmt.Errorf("创建视图 %s 失败: %v", name, err)
			}
			fmt.Printf("成功创建视图: %s\n", name)
		}
	}

	return nil
}
