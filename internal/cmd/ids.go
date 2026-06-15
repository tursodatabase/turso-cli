package cmd

import (
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func tryResolveOrgID(client *turso.Client) (string, error) {
	slug := client.Org
	if orgs := getOrgsCache(); orgs != nil {
		if id := findOrgID(orgs, slug); id != "" {
			return id, nil
		}
	}
	orgs, err := client.Organizations.List()
	if err != nil {
		return "", err
	}
	setOrgsCache(orgs)
	return findOrgID(orgs, slug), nil
}

func findOrgID(orgs []turso.Organization, slug string) string {
	for _, o := range orgs {
		if o.Slug == slug || (slug == "" && o.Type == "personal") {
			return o.ID
		}
	}
	return ""
}

func tryResolveGroupID(client *turso.Client, name string) (string, error) {
	if groups := getGroupsCache(client.Org); groups != nil {
		for _, g := range groups {
			if g.Name == name && g.UUID != "" {
				return g.UUID, nil
			}
		}
	}
	group, err := client.Groups.Get(name)
	if err != nil {
		return "", err
	}
	mergeGroupIntoCache(client.Org, group)
	return group.UUID, nil
}

func mergeGroupIntoCache(org string, group turso.Group) {
	groups := getGroupsCache(org)
	replaced := false
	for i := range groups {
		if groups[i].Name == group.Name {
			groups[i] = group
			replaced = true
			break
		}
	}
	if !replaced {
		groups = append(groups, group)
	}
	setGroupsCache(org, groups)
}

func tryResolveDbID(client *turso.Client, name string) (string, error) {
	if dbs := getDatabasesCache(); dbs != nil {
		for _, db := range dbs {
			if db.Name == name && db.ID != "" {
				return db.ID, nil
			}
		}
	}
	db, err := client.Databases.Get(name)
	if err != nil {
		return "", err
	}
	mergeDatabaseIntoCache(db)
	return db.ID, nil
}

func mergeDatabaseIntoCache(db turso.Database) {
	dbs := getDatabasesCache()
	replaced := false
	for i := range dbs {
		if dbs[i].Name == db.Name {
			dbs[i] = db
			replaced = true
			break
		}
	}
	if !replaced {
		dbs = append(dbs, db)
	}
	setDatabasesCache(dbs)
}
