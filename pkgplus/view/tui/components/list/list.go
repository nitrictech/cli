package list

type ListItem interface {
	GetItemValue() string
	GetItemDescription() string
}

type StringItem struct {
	Value string
}

func (s StringItem) GetItemValue() string {
	return s.Value
}

func (s StringItem) GetItemDescription() string {
	return ""
}

func StringsToListItems(strings []string) []ListItem {
	items := make([]ListItem, len(strings))
	for i, str := range strings {
		items[i] = StringItem{Value: str}
	}
	return items
}
