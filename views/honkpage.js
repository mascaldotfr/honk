{{ $BonkCSRF := .HonkCSRF }}
function encode(hash) {
        var s = []
        for (var key in hash) {
                var val = hash[key]
                s.push(escape(key) + "=" + escape(val))
        }
        return s.join("&")
}
function post(url, data) {
	var x = new XMLHttpRequest()
	x.open("POST", url)
	x.setRequestHeader("Content-Type", "application/x-www-form-urlencoded")
	x.send(data)
}
function get(url, whendone) {
	var x = new XMLHttpRequest()
	x.open("GET", url)
	x.responseType = "document"
	x.onload = function() { whendone(x) }
	x.send()
}
function bonk(el, xid) {
	el.innerHTML = "bonked"
	el.disabled = true
	post("/bonk", "CSRF={{ $BonkCSRF }}&xid=" + escape(xid))
}
function unbonk(el, xid) {
	el.innerHTML = "unbonked"
	el.disabled = true
	post("/zonkit", "CSRF={{ $BonkCSRF }}&wherefore=unbonk&what=" + escape(xid))
}
function muteit(el, convoy) {
	el.innerHTML = "muted"
	el.disabled = true
	post("/zonkit", "CSRF={{ $BonkCSRF }}&wherefore=zonvoy&what=" + escape(convoy))
	var els = document.querySelectorAll('article.honk')
	for (var i = 0; i < els.length; i++) {
		var e = els[i]
		if (e.getAttribute("data-convoy") == convoy) {
			e.remove()
		}
	}
}
function zonkit(el, xid) {
	el.innerHTML = "zonked"
	el.disabled = true
	post("/zonkit", "CSRF={{ $BonkCSRF }}&wherefore=zonk&what=" + escape(xid))
	var p = el
	while (p && p.tagName != "ARTICLE") {
		p = p.parentElement
	}
	if (p) {
		p.remove()
	}
}
function ackit(el, xid) {
	el.innerHTML = "acked"
	el.disabled = true
	post("/zonkit", "CSRF={{ $BonkCSRF }}&wherefore=ack&what=" + escape(xid))
}
function deackit(el, xid) {
	el.innerHTML = "deacked"
	el.disabled = true
	post("/zonkit", "CSRF={{ $BonkCSRF }}&wherefore=deack&what=" + escape(xid))
}
var topxid = { "{{ .PageName }}" : "{{ .TopXID }}" }
var honksforpage = { }
var thispagename = "{{ .PageName }}"
function fillinhonks(xhr) {
	var doc = xhr.responseXML
	topxid[thispagename] = doc.children[0].children[1].children[0].innerText
	var honks = doc.children[0].children[1].children[1].children
	var honksonpage = document.getElementById("honksonpage")
	var holder = honksonpage.children[0]
	var lenhonks = honks.length
	for (var i = honks.length; i > 0; i--) {
		holder.prepend(honks[i-1])
	}
	relinkconvoys()
	return lenhonks
}
function refreshhonks(btn) {
	btn.innerHTML = "refreshing"
	btn.disabled = true
	var args = { "page" : thispagename }
	args["topxid"] = topxid[thispagename]
	get("/hydra?" + encode(args), function(xhr) {
		var lenhonks = fillinhonks(xhr)
		btn.innerHTML = "refresh"
		btn.disabled = false
		btn.parentElement.children[1].innerHTML = " " + lenhonks + " new"
	})
}
function statechanger(evt) {
	var name = evt.state
	if (!name) {
		return
	}
	switchtopage(name)
}
function switchtopage(name, evt) {
	var honksonpage = document.getElementById("honksonpage")
	var holder = honksonpage.children[0]
	holder.remove()
	if (thispagename != "convoy") {
		honksforpage[thispagename] = holder
	}
	thispagename = name
	holder = honksforpage[name]
	if (holder) {
		honksonpage.prepend(holder)
	} else {
		honksonpage.prepend(document.createElement("div"))
		var args = { "page" : name }
		if (name == "convoy") {
			var c = evt.srcElement.text
			args["c"] = c
		} else {
			args["topxid"] = topxid[name]
		}
		get("/hydra?" + encode(args), fillinhonks)
	}
}
function pageswitcher(name) {
	return function(evt) {
		if (name == thispagename) {
			return false
		}
		switchtopage(name, evt)
		var url = evt.srcElement.href
		history.pushState(name, "some title", url)
		return false
	}
}
function relinkconvoys() {
	var els = document.getElementsByClassName("convoylink")
	for (var i = 0; i < els.length; i++) {
		els[i].onclick = pageswitcher("convoy")
	}
}
(function() {
	var el = document.getElementById("homelink")
	el.onclick = pageswitcher("home")
	var el = document.getElementById("atmelink")
	el.onclick = pageswitcher("atme")
	relinkconvoys()
	window.onpopstate = statechanger
	history.replaceState(thispagename, "some title", "")
})();
(function() {
	var el = document.getElementById("donkdescriptor")
	el.style.display = "none"
})();
function showhonkform(elem, rid, hname) {
	var form = document.getElementById("honkform")
	form.style = "display: block"
	if (elem) {
		form.remove()
		elem.parentElement.insertAdjacentElement('beforebegin', form)
	} else {
		elem = document.getElementById("honkformhost")
		elem.insertAdjacentElement('afterend', form)
	}
	var ridinput = document.getElementById("ridinput")
	var honknoise = document.getElementById("honknoise")
	if (rid) {
		ridinput.value = rid
		honknoise.value = "@" + hname + " "
	}
	document.getElementById("honknoise").focus()
}
function updatedonker() {
	var el = document.getElementById("donker")
	el.children[1].textContent = el.children[0].value.slice(-20)
	var el = document.getElementById("donkdescriptor")
	el.style.display = ""
}
