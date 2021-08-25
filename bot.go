package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	ttlcache "github.com/ReneKroon/ttlcache/v2"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Dialogue struct {
	Step  int
	Url   string
	Title string
	Tag   string
}

type Recipes struct {
	Recipes []Recipe `json:"recipes"`
}

type Recipe struct {
	Title    string   `json:"title"`
	SubTitle string   `json:"subTitle"`
	Url      string   `json:"url"`
	Filename string   `json:"filename"`
	Tags     []string `json:"tags"`
}

var notFound = ttlcache.ErrNotFound

func contains(n []int, number int) bool {
	for _, v := range n {
		if v == number {
			return true
		}
	}

	return false
}

func main() {
	ids := []int{}
	for _, i := range strings.Split(os.Getenv("IDS"), " ") {
		j, err := strconv.Atoi(string(i))
		if err != nil {
			panic(err)
		}
		ids = append(ids, j)
	}
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	var cache ttlcache.SimpleCache = ttlcache.NewCache()

	if err != nil {
		log.Fatal(err)
		return
	}

	var (
		tags        = &tb.ReplyMarkup{ResizeReplyKeyboard: true, OneTimeKeyboard: true}
		btnTag1     = tags.Text("Plat principal")
		btnTag2     = tags.Text("Dessert")
		btnTag3     = tags.Text("Gouter")
		btnTag4     = tags.Text("Accompagnement")
		btnTag5     = tags.Text("Entrée")
		selector    = &tb.ReplyMarkup{}
		btnCancel   = selector.Data("🚫", "cancel")
		btnValidate = selector.Data("👍", "validate")
	)
	tags.Reply(
		tags.Row(btnTag1),
		tags.Row(btnTag2),
		tags.Row(btnTag3),
		tags.Row(btnTag4),
		tags.Row(btnTag5),
	)
	selector.Inline(
		selector.Row(btnCancel, btnValidate),
	)

	b.Handle(tb.OnText, func(m *tb.Message) {
		if !contains(ids, m.Sender.ID) {
			b.Send(m.Sender, "There was en error with your id ("+strconv.Itoa(m.Sender.ID)+"). Please contact gp")
			return
		}

		val, err := cache.Get(strconv.Itoa(m.Sender.ID))

		if err == notFound {
			cache.Set(strconv.Itoa(m.Sender.ID), &Dialogue{Step: 0, Url: m.Text})
			b.Send(m.Sender, "*Url* : `"+m.Text+"`", tb.ModeMarkdownV2)
			b.Send(m.Sender, "Please send a title...")
			return
		}

		switch step := val.(*Dialogue).Step; step {
		case 0:
			cache.Set(strconv.Itoa(m.Sender.ID), &Dialogue{Step: 1, Url: val.(*Dialogue).Url, Title: m.Text})
			b.Send(m.Sender, "*Url* : `"+val.(*Dialogue).Url+"`\n*Title* : `"+m.Text+"`", tb.ModeMarkdownV2)
			b.Send(m.Sender, "Please choose a tag...", tags)
		}
	})

	var addTag = func(tag string, m *tb.Message) {
		if val, err := cache.Get(strconv.Itoa(m.Sender.ID)); err != notFound {
			cache.Set(strconv.Itoa(m.Sender.ID), &Dialogue{Step: 2, Url: val.(*Dialogue).Url, Title: val.(*Dialogue).Title, Tag: tag})
			b.Send(m.Sender, "*Url* : `"+val.(*Dialogue).Url+"`\n*Title* : `"+val.(*Dialogue).Title+"`\n*Tag* : `"+tag+"`", selector, tb.ModeMarkdownV2)
		} else {
			b.Send(m.Sender, "There was en error with adding tag. Please contact gp")
		}
	}

	var validate = func(c *tb.Callback) {
		if val, err1 := cache.Get(strconv.Itoa(c.Sender.ID)); err1 != notFound {
			byteValue, err2 := ioutil.ReadFile(os.Getenv("FILE"))
			if err2 != nil {
				b.Send(c.Sender, "There was en error with file opening. Please contact gp")
			} else {
				var recipes Recipes
				json.Unmarshal(byteValue, &recipes)
				json, _ := json.Marshal(Recipes{Recipes: append(recipes.Recipes, Recipe{Url: val.(*Dialogue).Url, Title: val.(*Dialogue).Title, Tags: []string{val.(*Dialogue).Tag}})})
				err3 := ioutil.WriteFile(os.Getenv("FILE"), json, 0644)
				if err3 != nil {
					b.Send(c.Sender, "There was en error with file writing. Please contact gp")
				} else {
					cache.Remove(strconv.Itoa(c.Sender.ID))
					b.Send(c.Sender, "Ok, new entry added")
				}
			}
		} else {
			b.Send(c.Sender, "There was en error. Please send an url to begin")
		}
		b.Respond(c, &tb.CallbackResponse{})

	}
	var cancel = func(c *tb.Callback) {
		cache.Remove(strconv.Itoa(c.Sender.ID))
		b.Respond(c, &tb.CallbackResponse{})
		b.Send(c.Sender, "Ok, please send another url to start again")
	}

	b.Handle(&btnTag1, func(m *tb.Message) { addTag("Plat principal", m) })
	b.Handle(&btnTag2, func(m *tb.Message) { addTag("Dessert", m) })
	b.Handle(&btnTag3, func(m *tb.Message) { addTag("Gouter", m) })
	b.Handle(&btnTag4, func(m *tb.Message) { addTag("Accompagnement", m) })
	b.Handle(&btnTag5, func(m *tb.Message) { addTag("Entrée", m) })
	b.Handle(&btnValidate, validate)
	b.Handle(&btnCancel, cancel)

	b.Handle("/start", func(m *tb.Message) {
		b.Send(m.Sender, "Please send an url to begin")
	})

	b.Start()
}
