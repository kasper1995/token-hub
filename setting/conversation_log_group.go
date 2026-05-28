package setting

import (
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

var conversationLogDisabledGroups = map[string]bool{}
var conversationLogDisabledGroupsMutex sync.RWMutex

func ConversationLogDisabledGroups2JSONString() string {
	conversationLogDisabledGroupsMutex.RLock()
	defer conversationLogDisabledGroupsMutex.RUnlock()

	groups := make([]string, 0, len(conversationLogDisabledGroups))
	for group, disabled := range conversationLogDisabledGroups {
		if disabled {
			groups = append(groups, group)
		}
	}
	sort.Strings(groups)

	jsonBytes, err := common.Marshal(groups)
	if err != nil {
		common.SysLog("error marshalling conversation log disabled groups: " + err.Error())
		return "[]"
	}
	return string(jsonBytes)
}

func CheckConversationLogDisabledGroupsByJSONString(jsonStr string) error {
	_, err := parseConversationLogDisabledGroups(jsonStr)
	return err
}

func UpdateConversationLogDisabledGroupsByJSONString(jsonStr string) error {
	groups, err := parseConversationLogDisabledGroups(jsonStr)
	if err != nil {
		return err
	}

	conversationLogDisabledGroupsMutex.Lock()
	defer conversationLogDisabledGroupsMutex.Unlock()
	conversationLogDisabledGroups = groups
	return nil
}

func IsConversationLogDisabledGroup(groupName string) bool {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return false
	}

	conversationLogDisabledGroupsMutex.RLock()
	defer conversationLogDisabledGroupsMutex.RUnlock()
	return conversationLogDisabledGroups[groupName]
}

func parseConversationLogDisabledGroups(jsonStr string) (map[string]bool, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return map[string]bool{}, nil
	}

	var list []string
	if err := common.Unmarshal([]byte(jsonStr), &list); err != nil {
		return nil, err
	}

	groups := make(map[string]bool, len(list))
	for _, group := range list {
		group = strings.TrimSpace(group)
		if group != "" {
			groups[group] = true
		}
	}
	return groups, nil
}
