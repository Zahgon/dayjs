English | [简体中文](./docs/zh-cn/README.zh-CN.md) | [日本語](./docs/ja/README-ja.md) | [Português Brasileiro](./docs/pt-br/README-pt-br.md) | [한국어](./docs/ko/README-ko.md) | [Español (España)](./docs/es-es/README-es-es.md) | [Русский](./docs/ru/README-ru.md) | [Türkçe](./docs/tr/README-tr.md) | [සිංහල](./docs/si/README-si.md) | [עברית](./docs/he/README-he.md)

<p align="center"><a href="https://day.js.org/" target="_blank" rel="noopener noreferrer"><img width="550"
                                                                             src="https://user-images.githubusercontent.com/17680888/39081119-3057bbe2-456e-11e8-862c-646133ad4b43.png"
                                                                             alt="Day.js" /></a></p>
<p align="center">Fast <b>2kB</b> alternative to Moment.js with the same modern API</p>
<p align="center">
    <a href="https://bundlephobia.com/package/dayjs"><img
            src="https://img.shields.io/bundlephobia/minzip/dayjs?style=flat-square&color=%2345cc11"
            alt="Gzip Size"></a>
    <a href="https://www.npmjs.com/package/dayjs"><img src="https://img.shields.io/npm/v/dayjs.svg?style=flat-square&colorB=51C838"
                                                       alt="NPM Version"></a>
    <a href="https://github.com/iamkun/dayjs/actions/workflows/check.yml"><img
            src="https://img.shields.io/github/actions/workflow/status/iamkun/dayjs/check.yml?style=flat-square" alt="Build Status"></a>
    <a href="https://codecov.io/gh/iamkun/dayjs"><img
            src="https://img.shields.io/codecov/c/github/iamkun/dayjs/master.svg?style=flat-square" alt="Codecov"></a>
    <a href="https://github.com/iamkun/dayjs/blob/master/LICENSE"><img
            src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square" alt="License"></a>
    <br>
    <a href="https://saucelabs.com/u/dayjs">
        <img width="750" src="https://user-images.githubusercontent.com/17680888/40040137-8e3323a6-584b-11e8-9dba-bbe577ee8a7b.png" alt="Sauce Test Status">
    </a>
</p>

> Day.js is a minimalist date library that parses, validates, manipulates, and displays dates and times with a largely Moment.js-compatible API. This is Day.js for Go: the core, all 37 plugins and all 143 locales, giving the same results as Day.js 1.11.23 on Node.js 24, down to how JavaScript's `Date` parses strings and resolves local times and how `Intl` reports time zones.

```go
dayjs.New().StartOf("month").Add(1, "day").Set("year", 2018).Format("YYYY-MM-DD HH:mm:ss")
```

* 🕒 Familiar Moment.js API & patterns
* 💪 Immutable
* 🔥 Chainable
* 🌐 I18n support
* 📦 No dependencies: time zone data is embedded

---

## Getting Started

### Documentation

You can find more details, API, and other docs on [day.js.org](https://day.js.org/) website. The Go API is the JavaScript one, mapped mechanically: `dayjs(...)` is `dayjs.New(...)`, `dayjs.utc(...)` is `dayjs.Utc(...)`, methods start with an upper-case letter, and a method that reads a value without arguments and sets it with them is two methods, `Year()` and `SetYear(2020)`. Numbers are `float64`, `null` is `nil`, `undefined` is `dayjs.Undefined`, an object literal is `dayjs.Obj{"years", 1, "months", 2}`, and an exception is a panic carrying a `*dayjs.Error`.

### Installation

```console
go get github.com/iamkun/dayjs-go
```

The module needs Go 1.27 and nothing else.

### API

It's easy to use Day.js APIs to parse, validate, manipulate, and display dates and times.

```go
import dayjs "github.com/iamkun/dayjs-go"

dayjs.New("2018-08-08") // parse

dayjs.New().Format("{YYYY} MM-DDTHH:mm:ss SSS [Z] A") // display

dayjs.New().Set("month", 3).Month() // get & set

dayjs.New().Add(1, "year") // manipulate

dayjs.New().IsBefore(dayjs.New()) // query
```

### I18n

Day.js has great support for internationalization. A locale is loaded by calling its function in package `locale`, as `require('dayjs/locale/es')` loads it in JavaScript.

```go
import "github.com/iamkun/dayjs-go/locale"

locale.Es() // load a locale
dayjs.Locale("es") // use it globally

locale.ZhCn()
dayjs.New("2018-05-05").SetLocale("zh-cn").Format() // use it in one instance
```

### Plugin

A plugin is an independent module that can be added to Day.js to extend functionality or add new features.

```go
import "github.com/iamkun/dayjs-go/plugin"

dayjs.Extend(plugin.AdvancedFormat) // use a plugin

dayjs.New().Format("Q Do k kk X x") // more available formats
```

### Tests

`go test ./...` replays a battery of calls recorded from Day.js 1.11.23 on Node.js 24 in several time zones (`testdata/replay.jsonl`), each in a process of its own.

## Downloads
![Day.js Chart](./docs/assets/download.png)

## Sponsors

Support this project by becoming a sponsor. Your logo will show up here with a link to your website.

[[Become a sponsor via Github](https://github.com/sponsors/iamkun/)] [[Become a sponsor via OpenCollective](https://opencollective.com/dayjs#sponsor)]

<a href="https://github.com/ken-swyfft" target="_blank">
  <img width="70" src="https://avatars.githubusercontent.com/u/65305317?v=4">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://opencollective.com/sight-sound" target="_blank">
  <img width="70" src="https://images.opencollective.com/sight-sound/54f7220/logo/256.png?height=256">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://opencollective.com/filerev" target="_blank">
  <img width="70" src="https://images.opencollective.com/filerev/93a8f05/logo/256.png?height=256" />
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://opencollective.com/carboneio" target="_blank">
  <img width="70" src="https://images.opencollective.com/carboneio/fe2066c/logo/256.png?height=256">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://www.netrouting.com" target="_blank" alt="Netrouting Dedicated Servers">
  <img width="70" src="https://netrouting.com/wp-content/uploads/2026/04/Logo-netrouting.png">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://anonstories.com" target="_blank">
  <img alt="Instagram Story Viewer" width="70" src="https://avatars.githubusercontent.com/u/240702364?v=4">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://www.sayfone.com/call-rates " target="_blank">
  <img alt="Sayfone Cheap international calls" width="70" src="https://github.com/user-attachments/assets/68f91139-b190-421b-bcd7-43a8e3d84fe7">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://automatio.ai" target="_blank" alt="Automatio AI">
  <img width="70" src="https://avatars.githubusercontent.com/u/1984909?v=4" />
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://bestkru.com/" target="_blank">
  <img width="70" src="https://avatars.githubusercontent.com/u/159320286" alt="BestKru">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://handsontable.com/docs/react-data-grid/?utm_source=Dayjs_GH&utm_medium=sponsorship&utm_campaign=library_sponsorship" target="_blank">
  <img width="70" src="https://github.com/user-attachments/assets/426c3476-dc34-44d1-a904-ed58dbd20dd6">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://route4me.com/" target="_blank">
  <img width="70" src="https://github.com/user-attachments/assets/3fbc86c5-98a9-49c2-beae-1969026fcd76" alt="Route Optimizer and Route Planner Software">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://github.com/sentdm" target="_blank">
  <img width="70" src="https://avatars.githubusercontent.com/u/153308555?s=200&v=4">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://github.com/blacksandsmedia" target="_blank">
  <img width="70" src="https://avatars.githubusercontent.com/u/116517387?v=4">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://github.com/mvpsnet" target="_blank">
  <img width="70" src="https://avatars.githubusercontent.com/u/89784111?s=96&v=4">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://onlydoc.com/" target="_blank" alt><img alt="" width="70" src="https://github.com/user-attachments/assets/f5d5938e-48a1-420b-af72-c4012d3941c1" /></a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://unaimytext.com" target="_blank">
  <img width="70" src="https://github.com/user-attachments/assets/8510473a-c097-4bc9-8c61-b7515194f6f5">
</a>


## Contributors

This project exists thanks to all the people who contribute.

Please give us a 💖 star 💖 to support us. Thank you.

And thank you to all our backers! 🙏

<a href="https://opencollective.com/dayjs#backers" target="_blank"><img src="https://opencollective.com/dayjs/contributors.svg?width=890" /></a>

## License

Day.js is licensed under a [MIT License](./LICENSE).
