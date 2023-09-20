package main

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

var client = req.C().SetTimeout(time.Second * 5)

func main() {
	a := app.New()
	w := a.NewWindow("IP Location Finder")
	w.SetContent(makeUI())
	w.Resize(fyne.NewSize(w.Canvas().Size().Width, 650))
	w.ShowAndRun()
}

func makeUI() fyne.CanvasObject {
	header := canvas.NewText("IP Location Finder", theme.PrimaryColor())
	header.TextSize = 42
	header.Alignment = fyne.TextAlignCenter

	u, _ := url.Parse("https://github.com/able8/ip-location-finder")
	footer := widget.NewHyperlinkWithStyle("github.com/able8/ip-location-finder", u, fyne.TextAlignCenter, fyne.TextStyle{})

	ip := binding.NewString()
	input := widget.NewEntryWithData(ip)
	input.SetPlaceHolder("Input any IP address")
	input.Validator = func(s string) error {
		parsedIP := net.ParseIP(s)
		if parsedIP != nil {
			return nil
		}
		return errors.New("wrong IP")
	}

	var data = []ipInfo{}
	var list *widget.List
	list = widget.NewList(
		func() int {
			return len(data)
		},
		func() fyne.CanvasObject {
			url := widget.NewHyperlinkWithStyle("URL: ", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			countryName := widget.NewLabelWithStyle("Country: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			city := widget.NewLabelWithStyle("City: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			isp := widget.NewLabelWithStyle("ISP: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			org := widget.NewLabelWithStyle("ORG: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			return container.NewVBox(
				url, container.NewHBox(countryName, city), isp, org,
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			objs := o.(*fyne.Container).Objects
			objs[0].(*widget.Hyperlink).SetText("URL: " + data[i].url)
			objs[0].(*widget.Hyperlink).SetURLFromString(data[i].url)
			objs[1].(*fyne.Container).Objects[0].(*widget.Label).SetText("Country: " + data[i].countryName)
			objs[1].(*fyne.Container).Objects[1].(*widget.Label).SetText("City: " + data[i].city)
			if data[i].isp == data[i].org || strings.EqualFold(data[i].isp, data[i].org) {
				objs[2].(*widget.Label).SetText("ISP/ORG: " + data[i].isp)
				objs[3].Hide()
				list.SetItemHeight(i, list.MinSize().Height/4*3)
			} else {
				objs[2].(*widget.Label).SetText("ISP: " + data[i].isp)
				objs[3].(*widget.Label).SetText("ORG: " + data[i].org)
			}
		})

	search := widget.NewButtonWithIcon("Look Up", theme.SearchIcon(), func() {
		ip := input.Text
		if ip == "" {
			ip = "8.8.8.8"
		}
		data = nil
		go func() {
			for info := range findIPResults(ip) {
				if info.countryName != "" {
					data = append(data, info)
					list.Refresh()
				}
			}
		}()
	})
	search.Importance = widget.HighImportance

	return container.NewBorder(container.NewVBox(header, container.NewGridWithRows(2, input, search)), footer, nil, nil, list)
}

type ipInfo struct {
	ip          string
	countryName string
	city        string
	isp         string
	org         string
	url         string
}

func find(ip string, provider ipInfo) ipInfo {
	url := strings.ReplaceAll(provider.url, "1.1.1.1", ip)
	resp, err := client.R().Get(url)
	if err != nil {
		return ipInfo{}
	}

	if resp.StatusCode != http.StatusOK {
		return ipInfo{}
	}

	data := gjson.ParseBytes(resp.Bytes())
	return ipInfo{
		ip:          ip,
		countryName: data.Get(provider.countryName).String(),
		city:        data.Get(provider.city).String(),
		isp:         data.Get(provider.isp).String(),
		org:         data.Get(provider.org).String(),
		url:         url,
	}
}

func findIPResults(ip string) chan ipInfo {
	resultChan := make(chan ipInfo, 2)
	for i := range providers {
		go func(i int) {
			resultChan <- find(ip, providers[i])
		}(i)
	}
	return resultChan
}

var providers = []ipInfo{
	{
		url:         "https://ipapi.co/1.1.1.1/json/",
		countryName: "country_name",
		city:        "city",
		isp:         "org",
		org:         "org",
	},
	{
		url:         "http://ip-api.com/json/1.1.1.1",
		countryName: "country",
		city:        "city",
		isp:         "isp",
		org:         "org",
	},
	{
		url:         "http://ipinfo.io/1.1.1.1/json",
		countryName: "country",
		city:        "city",
		isp:         "org",
		org:         "org",
	},
	{
		url:         "http://ip.qste.com/json?ip=1.1.1.1",
		countryName: "country",
		city:        "city",
		isp:         "isp",
		org:         "org",
	},
	// {
	// 	url:         "https://qifu-api.baidubce.com/ip/geo/v1/district?ip=1.1.1.1",
	// 	countryName: "data.country",
	// 	city:        "data.city",
	// 	isp:         "data.isp",
	// 	org:         "data.owner",
	// },
}
