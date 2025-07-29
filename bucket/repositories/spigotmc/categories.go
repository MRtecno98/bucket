// Package spigotmc provides additional functionality for the SpigotMC repository.
package spigotmc

import (
	"errors"
	"slices"

	"github.com/MRtecno98/bucket/bucket/platforms"
	"github.com/sunxyw/go-spiget/spiget"
)

const (
	BungeeSpigot = 2 + iota
	BungeeProxy
	Spigot
	Transportation5
	Chat6
	Utilities7
	Misc8
	Libraries9
	Transportation10
	Chat11
	Utilities12
	Misc13
	Chat14
	Utilities15
	Misc16
	Fun
	WorldManagement
	Standalone
	Premium
	Universal
	Mechanics
	Economy
	GameMode
	Skript
	Libraries26
	Web
	NoRating
	DataPack
)

type Category struct {
	ID            int
	Subcategories []Category

	compatiblePlatforms []string
}

var Categories = []Category{
	{ID: BungeeSpigot, Subcategories: []Category{
		{ID: Transportation5}, {ID: Chat6}, {ID: Utilities7}, {ID: Misc8}, {ID: Universal},
	}, compatiblePlatforms: []string{platforms.BungeeTypePlatform.Name, platforms.SpigotTypePlatform.Name}},

	{ID: BungeeProxy, Subcategories: []Category{
		{ID: Libraries9}, {ID: Transportation10}, {ID: Chat11}, {ID: Utilities12}, {ID: Misc13}, {ID: Universal},
	}, compatiblePlatforms: []string{platforms.BungeeTypePlatform.Name}},

	{ID: Spigot, Subcategories: []Category{
		{ID: Chat14}, {ID: Utilities15}, {ID: Misc16}, {ID: Fun}, {ID: WorldManagement},
		{ID: Mechanics}, {ID: Economy}, {ID: GameMode} /* {ID: Skript}, */, {ID: Libraries26},
		{ID: NoRating}, {ID: Universal},
	}, compatiblePlatforms: []string{platforms.SpigotTypePlatform.Name,
		platforms.PaperTypePlatform.Name, platforms.PurpurTypePlatform.Name}},

	{ID: Universal, Subcategories: []Category{
		{ID: BungeeSpigot}, {ID: BungeeProxy}, {ID: Spigot}, {ID: Standalone},
	}},

	{ID: Standalone, Subcategories: []Category{}},
	{ID: Premium, Subcategories: []Category{}},
	{ID: Web, Subcategories: []Category{}},
	{ID: DataPack, Subcategories: []Category{}},
}

func GetCategory(cat spiget.Category) (*Category, error) {
	for _, c := range AllCategories() {
		if c.ID == cat.ID {
			return &c, nil
		}
	}

	return nil, errors.New("category not found")
}

func AllCategories() []Category {
	var categories []Category
	for _, cat := range Categories {
		categories = append(categories, cat)
		categories = append(categories, cat.Subcategories...) // Works because there's only 1 level of subcategories
	}

	return categories
}

func (c Category) Compatible(platform string) bool {
	return slices.Contains(c.compatiblePlatforms, platform)
}

func (c *Category) CompatiblePlatforms() []string {
	var platforms []string
	for _, cat := range c.Parents() {
		platforms = append(platforms, cat.compatiblePlatforms...)
	}

	slices.Sort(platforms)
	return slices.Compact(platforms)
}

func (c Category) Parents() []Category {
	var parents []Category
	for _, cat := range Categories {
		for _, sub := range cat.Subcategories {
			if sub.ID == c.ID {
				parents = append(parents, cat)
			}
		}
	}

	return parents
}
