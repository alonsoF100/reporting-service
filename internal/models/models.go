package models

// TODO добавить модели данных
type DeviceMessage struct {
    Number      int    `json:"number"`       // #номер
    Mqtt        string `json:"mqtt"`         // mqtt
    Invid       string `json:"invid"`        // инвентарный номер
    UnitGUID    string `json:"unit_guid"`    // гуид устройства
    MessageID   string `json:"message_id"`   // id сообщения
    MessageText string `json:"message_text"` // текст сообщения
    Context     string `json:"context"`      // среда
    MessageClass string `json:"message_class"` // класс сообщения [alarm,warning,info,event,comand,waiting,working]
    Level       int    `json:"level"`        // уровень сообщения [int]
    Area        string `json:"area"`         // зона переменных HR,IR.I,C
    Address     string `json:"address"`      // адрес переменной в контроллере
}

type ParseResult struct {
	FileName string `json:"file_name"`
	Messages []DeviceMessage `json:"messages"`
}