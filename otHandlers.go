package orgtracker

import (
	sl "github.com/djosephsen/slacker/slackerlib"
	"encoding/json"
	"regexp"
	"fmt"
	"strings"
)

const ORGS string = `OT:ORGS`

type Org struct{
	Name string
	Members	map[string]sl.User
}

func getOrgs(bot *sl.Sbot) (map[string]Org, error){
// load in the orgs struct from brain
 	orgs := make(map[string]Org)
	brain := *bot.Brain
	if orgson, err := brain.Get(ORGS); err != nil{
		//if notFound,_ := regexp.MatchString(
		if err.Error() != `key OT:ORGS was not found`{
			return orgs, err
		}
	}else if err := json.Unmarshal(orgson, &orgs); err != nil{
		return orgs, err
	}
	return orgs, nil
}

func setOrgs(bot *sl.Sbot, orgs map[string]Org) error{
// write the orgs struct to brain
	brain := *bot.Brain
	if orgson, err := json.Marshal(&orgs); err !=nil{
		return err
	}else if err = brain.Set(ORGS, orgson); err != nil{
		return err
	}
	return nil
}

var QueryPeeps = sl.MessageHandler{
	Name: `OrgTracker, Query Peeps`,
	Usage:`"<botname> who is from <orgname>" returns the users from <orgname>`,
	Method: `RESPOND`,
	Pattern: `(?i)who is from (\w+)`,
	Run:	func(e *sl.Event, match []string){
		orgName := match[1]
		orgID := strings.ToLower(orgName)
		orgs, err := getOrgs(e.Sbot) 
		if err != nil{
			e.Respond(fmt.Sprintf("ack! I couldn't load my orgs struct! %s", err))
			sl.Logger.Debug(err)
			return
		}

		if org, exists := orgs[orgID]; exists{
			users := fmt.Sprintf("Members from %s:\n",orgName)
			for _,peep := range org.Members{	
				users = fmt.Sprintf("%s\n%s", users, peep.Name)
			}
			e.Respond(users)	
			return
		}else{
			e.Respond(fmt.Sprintf("sorry, no org called: %s", orgName))
			return
		}
	},
}

var ManageOrg = sl.MessageHandler{
	Name: `OrgTracker: Manage Org`,
	Usage:`"<botname> (add|delete) org <orgname>" adds or deletes an org called <orgname>`,
	Method: `RESPOND`,
	Pattern: `(?i)(add|delete) org (\w+)`,
	Run:	func(e *sl.Event, match []string){
		cmd:=match[1]
		orgName:=match[2]
		orgID := strings.ToLower(orgName)
		orgs, err := getOrgs(e.Sbot) 
		if err != nil{
			e.Respond(fmt.Sprintf("ack! I couldn't load my orgs struct! %s", err))
			sl.Logger.Debug(err)
			return
		}

		if isAdd,_ := regexp.MatchString( `(?i)add`, cmd); isAdd{
			newOrg:=Org{
				Name:	strings.ToLower(orgName),
				Members: make(map[string]sl.User),
			}
			orgs[orgID] = newOrg
			if err := setOrgs(e.Sbot, orgs); err != nil{
				e.Reply(fmt.Sprintf("sorry I couldn't add %s because my brain says: %s", orgName, err))
				return
			}
			e.Reply(fmt.Sprintf("sure thing. Org: %s added", orgName))
		}

		if isDelete,_ := regexp.MatchString( `(?i)delete`, cmd); isDelete{
			delete(orgs,orgID)
			if err := setOrgs(e.Sbot, orgs); err != nil{
				e.Reply(fmt.Sprintf("blerg, couldn't delete %s because my brain says: %s", orgName, err))
				return
			}

			e.Reply(fmt.Sprintf("sure thing. Org: %s deleted", orgName))
		}
	},
}

var JoinOrg = sl.MessageHandler{
	Name: `OrgTracker: Join Org`,
	Usage:`"<botname> (join|leave) org <orgname>" adds or removes you to/from <orgname>`,
	Method: `RESPOND`,
	Pattern: `(?i)(join|leave) org (\w+)`,
	Run:	func(e *sl.Event, match []string){
		cmd:=match[1]
		orgName:=match[2]
		orgID := strings.ToLower(orgName)

		orgs, err := getOrgs(e.Sbot) 
		if err != nil{
			e.Respond(fmt.Sprintf("ack! I couldn't load my orgs struct! %s", err))
			sl.Logger.Debug(err)
			return
		}

		if isJoin,_ := regexp.MatchString( `(?i)join`, cmd); isJoin{
			if org, exists := orgs[orgID]; exists{
				user:=e.Sbot.Meta.GetUser(e.User)
				if _,exists := org.Members[user.ID]; exists{
					e.Reply(fmt.Sprintf("user: %s already belongs to %s (sorry)",user.Name, orgName))
				}else{
					org.Members[user.ID] = *user
					if err := setOrgs(e.Sbot, orgs); err != nil{
						e.Reply(fmt.Sprintf("derp.. I couldn't add %s. Brain trouble: %s", user.Name, err))
						return
					}
					e.Reply(fmt.Sprintf("OK! user: %s now belongs to %s",user.Name, orgName))
					return
				}
			}else{
				e.Reply(fmt.Sprintf("No such org: %s (sorry)",orgName))
			}
		}
		if isLeave,_ := regexp.MatchString( `(?i)leave`, cmd); isLeave{
			if org,exists := orgs[orgID]; exists{ 
				user:=e.Sbot.Meta.GetUser(e.User)
				delete (org.Members,user.ID)
				e.Reply(fmt.Sprintf("OK! %s is no longer in %s",user.Name, orgName))
			}else{
				e.Reply(fmt.Sprintf("No such org: %s (sorry)",orgName))
			}
		}
	},
}
