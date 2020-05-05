package store

import (
	"github.com/jinzhu/gorm"
	"github.com/raahii/golang-grpc-realworld-example/model"
)

// ArticleStore is data access struct for user
type ArticleStore struct {
	db *gorm.DB
}

// NewArticleStore returns a new ArticleStore
func NewArticleStore(db *gorm.DB) *ArticleStore {
	return &ArticleStore{
		db: db,
	}
}

// GetByID finds an article from id
func (s *ArticleStore) GetByID(id uint) (*model.Article, error) {
	var m model.Article
	err := s.db.Preload("Tags").Preload("Author").Find(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create creates an article
func (s *ArticleStore) Create(m *model.Article) error {
	return s.db.Create(&m).Error
}

// Update updates an article
func (s *ArticleStore) Update(m *model.Article) error {
	return s.db.Model(&m).Update(&m).Error
}

// GetArticles get global articles
func (s *ArticleStore) GetArticles(tagName, username string, favoritedBy *model.User, limit, offset int64) ([]model.Article, error) {
	d := s.db.Preload("Author")

	// author query (has one)
	if username != "" {
		d = d.Joins("join users on articles.user_id = users.id").
			Where("users.username = ?", username)
	}

	// tag query (many to many)
	if tagName != "" {
		d = d.Joins(
			"join article_tags on articles.id = article_tags.article_id "+
				"join tags on tags.id = article_tags.tag_id").
			Where("tags.name = ?", tagName)
	}

	// favorited query
	if favoritedBy != nil {
		rows, err := s.db.Select("article_id").
			Table("favorite_articles").
			Where("user_id = ?", favoritedBy.ID).
			Offset(offset).Limit(limit).Rows()
		if err != nil {
			return []model.Article{}, err
		}
		defer rows.Close()

		var ids []uint
		for rows.Next() {
			var id uint
			rows.Scan(&id)
			ids = append(ids, id)
		}
		d = d.Where("id in (?)", ids)
	}

	// offset query, limit query
	d = d.Offset(offset).Limit(limit)

	var as []model.Article
	err := d.Find(&as).Error

	return as, err
}

// GetFeedArticles returns following users' articles
func (s *ArticleStore) GetFeedArticles(userIDs []uint, limit, offset int64) ([]model.Article, error) {
	d := s.db.Preload("Author").
		Where("user_id in (?)", userIDs)

	// offset query, limit query
	d = d.Offset(offset).Limit(limit)

	var as []model.Article
	err := d.Find(&as).Error

	return as, err
}

// Delete deletes an article
func (s *ArticleStore) Delete(m *model.Article) error {
	return s.db.Delete(m).Error
}

// IsFavorited returns whether the article is favorited by the user
func (s *ArticleStore) IsFavorited(a *model.Article, u *model.User) (bool, error) {
	var count int
	err := s.db.Table("favorite_articles").
		Where("article_id = ? AND user_id = ?", a.ID, u.ID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// AddFavorite favorite an article
func (s *ArticleStore) AddFavorite(a *model.Article, u *model.User) error {
	tx := s.db.Begin()

	err := tx.Model(a).Association("FavoritedUsers").
		Append(u).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(a).
		Update("favorites_count", gorm.Expr("favorites_count + ?", 1)).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	a.FavoritesCount++

	return nil
}
