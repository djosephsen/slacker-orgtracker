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

var OrgTracker = sl.MessageHandler{
	Name: `OrgTracker`,
	Usage:`<botname> [add|delete|join|leave|list] org <orgname>`,
	Method: `RESPOND`,
	Pattern: `(?i)(add|delete|join|leave|list) orgs* *(\w*)`,
	Run:	func(e *sl.Event, match []string){
		var orgName, orgID string
		cmd := match[1]
		if len(match) >= 2{
			orgName = match[2]
			orgID = strings.ToLower(orgName)
		}
		orgs, err := getOrgs(e.Sbot) 
		if err != nil{
			e.Respond(fmt.Sprintf("ack! I couldn't load my orgs struct! %s", err))
			sl.Logger.Debug(err)
			return
		}
		if isAdd,_ := regexp.MatchString( `(?i)add`, cmd); isAdd{
			if err := addOrg(e.Sbot,orgs,orgID); err!=nil{
				e.Reply(fmt.Sprintf("sorry I couldn't add %s because: %s", orgName, err))
				return
			}
			e.Reply(fmt.Sprintf("sure thing. Org: %s added", orgName))
			return
		}else if isDel,_ := regexp.MatchString( `(?i)delete`, cmd); isDel{
			if err := deleteOrg(e.Sbot,orgs,orgID); err!=nil{
				e.Reply(fmt.Sprintf("gar. %s", err))
				return
			}
			e.Reply(fmt.Sprintf("Ok. Org: %s deleted", orgName))
			return
		}else if isJoin,_ := regexp.MatchString( `(?i)join`, cmd); isJoin{
			if _, exists := orgs[orgID]; !exists{
				e.Reply(fmt.Sprintf("(Creating new org: %s first)",orgName))
				if err := addOrg(e.Sbot,orgs,orgID); err!=nil{
					e.Reply(fmt.Sprintf("sorry I couldn't add %s because: %s", orgName, err))
					return
				}
			}
			if err := joinOrg(e,orgs,orgID); err!=nil{
					e.Reply(fmt.Sprintf("derp. %s", err))
					return
			}
			user:=e.Sbot.Meta.GetUser(e.User)
			e.Reply(fmt.Sprintf("OK! user: %s now belongs to %s",user.Name, orgName))
			return
		}else if isLeave,_ := regexp.MatchString( `(?i)leave`, cmd); isLeave{
			if err := leaveOrg(e,orgs,orgID); err!=nil{
					e.Reply(fmt.Sprintf("bleh. %s", err))
					return
			}
			user:=e.Sbot.Meta.GetUser(e.User)
			e.Reply(fmt.Sprintf("OK! removed user: %s from org: %s",user.Name, orgName))
			return
		}else if isList,_ := regexp.MatchString( `(?i)list`, cmd); isList{
			if reply, err := listOrg(orgs,orgID); err!=nil{
				e.Reply(fmt.Sprintf("sorry. %s", err))
				return
			}else{
				e.Reply(reply)
				return
			}
		}
	},
}

func listOrg(orgs map[string]Org, orgID string) (string, error){
	if orgID != ``{
		if org, exists := orgs[orgID]; exists{
			users := fmt.Sprintf("Members from %s:\n",orgID)
			for _,peep := range org.Members{	
				users = fmt.Sprintf("%s\n%s", users, peep.Name)
			}
			return users, nil
		}else{
			return ``,fmt.Errorf("no org called: %s", orgID)
		}
	}else{
		reply:=`Existing Orgs:`
		for orgid,org := range orgs{
			reply=fmt.Sprintf("%s\n%s, (%d members)",reply, orgid, len(org.Members))
		}
		return reply,nil
	}
}

func listFrom(bot *sl.Sbot, orgs map[string]Org, usr string) (string, error){
	if user := bot.Meta.GetUserByName(usr); user != nil{
		reply:=fmt.Sprintf("The following orgs are lucky enough to have %s's help:",user.Name)
		for _, org := range orgs{
			if _,isAMember := org.Members[user.ID]; isAMember{
				reply=fmt.Sprintf("%s\n%s",reply,org.Name)
			}
		}
		return reply,nil
	}else{
		return ``,fmt.Errorf("I don't know anyone named: %s", usr)
	}
}
					

func addOrg(bot *sl.Sbot, orgs map[string]Org, orgID string) error{
	newOrg:=Org{
		Name:	orgID,
		Members: make(map[string]sl.User),
	}
	orgs[orgID] = newOrg
	if err := setOrgs(bot, orgs); err != nil{
		return fmt.Errorf("I couldn't add %s because my brain says: %s", orgID, err)
	}
	return nil
}

func deleteOrg(bot *sl.Sbot, orgs map[string]Org, orgID string) error{
	delete(orgs,orgID)
	if err := setOrgs(bot, orgs); err != nil{
		return fmt.Errorf("I couldn't delete %s because my brain says: %s", orgID, err)
	}
	return nil
}

func joinOrg(e *sl.Event, orgs map[string]Org, orgID string) error{
	user := e.Sbot.Meta.GetUser(e.User)
	org := orgs[orgID]
	if _,exists := org.Members[user.ID]; exists{
		return fmt.Errorf("user: %s already belongs to %s (sorry)",user.Name, orgID)
	}else{
		org.Members[user.ID] = *user
		if err := setOrgs(e.Sbot, orgs); err != nil{
			return fmt.Errorf("I couldn't add %s. Brain trouble: %s", user.Name, err)
		}
		return nil
	}
}

func leaveOrg(e *sl.Event, orgs map[string]Org, orgID string) error{
	if org,exists := orgs[orgID]; exists{ 
		user:=e.Sbot.Meta.GetUser(e.User)
		delete (org.Members,user.ID)
		if err := setOrgs(e.Sbot, orgs); err != nil{
			return fmt.Errorf("I couldn't delete %s. Brain trouble: %s", user.Name, err)
		}
		return nil
	}else{
		return fmt.Errorf("No such org: %s (sorry)",orgID)
	}
}

func addUserToOrg(e *sl.Event, orgs map[string]Org, usr string, orgID string) error{
	user := e.Sbot.Meta.GetUserByName(usr)
	if user == nil{
		return fmt.Errorf("I couldn't find a user named: %s", usr)
	}
	org := orgs[orgID]
	if _,exists := org.Members[user.ID]; exists{
		return fmt.Errorf("user: %s already belongs to %s (sorry)",user.Name, orgID)
	}else{
		org.Members[user.ID] = *user
		if err := setOrgs(e.Sbot, orgs); err != nil{
			return fmt.Errorf("I couldn't add %s. Brain trouble: %s", user.Name, err)
		}
		return nil
	}
}

func deleteUserFromOrg(e *sl.Event, orgs map[string]Org, usr string, orgID string) error{
	if org,exists := orgs[orgID]; exists{ 
		if user := e.Sbot.Meta.GetUserByName(usr); user == nil{
			delete (org.Members,user.ID)
			if err := setOrgs(e.Sbot, orgs); err != nil{
				return fmt.Errorf("I couldn't delete %s. Brain trouble: %s", user.Name, err)
			}
			return nil
		}else{
			return fmt.Errorf("No such user: %s (sorry)",usr)
		}
	}else{
		return fmt.Errorf("No such org: %s (sorry)",orgID)
	}
}

func getOrgs(bot *sl.Sbot) (map[string]Org, error){
// load in the orgs struct from brain
 	orgs := make(map[string]Org)
	brain := *bot.Brain
	if orgson, err := brain.Get(ORGS); err != nil{
		if notFound,_ := regexp.MatchString(`ORGS.*not.*found`,err.Error()); !notFound{
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

var WhoIsFrom = sl.MessageHandler{
	Name: `OrgTracker: Who-IS-FROM`,
	Usage:`"<botname> who is from <org>" :: lists members who are from org`,
	Method: `RESPOND`,
	Pattern: `(?i)who is from (\w*)`,
	Run:	func(e *sl.Event, match []string){
		orgName := match[1]
		orgID := strings.ToLower(orgName)
		orgs, err := getOrgs(e.Sbot) 
		if err != nil{
			e.Respond(fmt.Sprintf("ack! I couldn't load my orgs struct! %s", err))
			sl.Logger.Debug(err)
			return
		}
		if reply, err := listOrg(orgs,orgID); err!=nil{
			e.Reply(fmt.Sprintf("sorry. %s", err))
			return
		}else{
			e.Reply(reply)
			return
		}
	},
}

var WhereIsFrom = sl.MessageHandler{
	Name: `OrgTracker: Where-IS-FROM`,
	Usage:`"<botname> where is <user> from?" :: lists orgs that user belongs to`,
	Method: `RESPOND`,
	Pattern: `(?i)where is (\w*) from`,
	Run:	func(e *sl.Event, match []string){
		userName := match[1]
		orgs, err := getOrgs(e.Sbot) 
		if err != nil{
			e.Respond(fmt.Sprintf("ack! I couldn't load my orgs struct! %s", err))
			sl.Logger.Debug(err)
			return
		}
		if reply, err := listFrom(e.Sbot,orgs,userName); err!=nil{
			e.Reply(fmt.Sprintf("sorry. %s", err))
			return
		}else{
			e.Reply(reply)
			return
		}
	},
}


var OTUserManage = sl.MessageHandler{
	Name: `OrgTracker: User Manage`,
	Usage:`"<botname> (add|delete) <user> (to|from) org <org>" :: adds or deletes person to org`,
	Method: `RESPOND`,
	Pattern: `(?i)(add|delete) (\w*) (to|from) org (\w*)`,
	Run:	func(e *sl.Event, match []string){
		cmd := match[1]
		userName := match[2]
		orgName := match[4]
		orgID := strings.ToLower(orgName)
		orgs, err := getOrgs(e.Sbot) 
		if err != nil{
			e.Respond(fmt.Sprintf("ack! I couldn't load my orgs struct! %s", err))
			sl.Logger.Debug(err)
			return
		}
		if isAdd,_ := regexp.MatchString( `(?i)add`, cmd); isAdd{
			if _, exists := orgs[orgID]; !exists{
				e.Reply(fmt.Sprintf("(Creating new org: %s first)",orgName))
				if err := addOrg(e.Sbot,orgs,orgID); err!=nil{
					e.Reply(fmt.Sprintf("sorry I couldn't add %s because: %s", orgName, err))
					return
				}
			}
			if err := addUserToOrg(e,orgs,userName,orgID); err!=nil{
				e.Reply(fmt.Sprintf("sorry, %s", err))
				return
			}
			e.Reply(fmt.Sprintf("Ok. %s added to %s", userName, orgName))
			return
		}else if isDelete,_ := regexp.MatchString( `(?i)delete`, cmd); isDelete{
			if err := deleteUserFromOrg(e,orgs,userName,orgID); err!=nil{
				e.Reply(fmt.Sprintf("bummer. %s", err))
				return
			}
			e.Reply(fmt.Sprintf("Ok. %s deleted from %s", userName, orgName))
			return
		}else{
			sl.Logger.Error(`OT:: Got a wierd cmd in user manage: `, cmd)
		}
	},
}
