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
	post("/bonk", encode({"CSRF": csrftoken, "xid": xid}))
}
function unbonk(el, xid) {
	el.innerHTML = "unbonked"
	el.disabled = true
	post("/zonkit", encode({"CSRF": csrftoken, "wherefore": "unbonk", "what": xid}))
}
function muteit(el, convoy) {
	el.innerHTML = "muted"
	el.disabled = true
	post("/zonkit", encode({"CSRF": csrftoken, "wherefore": "zonvoy", "what": convoy}))
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
	post("/zonkit", encode({"CSRF": csrftoken, "wherefore": "zonk", "what": xid}))
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
	post("/zonkit", encode({"CSRF": csrftoken, "wherefore": "ack", "what": xid}))
}
function deackit(el, xid) {
	el.innerHTML = "deacked"
	el.disabled = true
	post("/zonkit", encode({"CSRF": csrftoken, "wherefore": "deack", "what": xid}))
}
function fillinhonks(xhr) {
	var doc = xhr.responseXML
	var stash = curpagestate.name + ":" + curpagestate.arg
	topxid[stash] = doc.children[0].children[1].children[0].innerText
	var honks = doc.children[0].children[1].children[1].children
	var honksonpage = document.getElementById("honksonpage")
	var holder = honksonpage.children[0]
	var lenhonks = honks.length
	for (var i = honks.length; i > 0; i--) {
		holder.prepend(honks[i-1])
	}
	relinklinks()
	return lenhonks
}
function hydrargs() {
	var name = curpagestate.name
	var arg = curpagestate.arg
	var args = { "page" : name }
	if (name == "convoy") {
		args["c"] = arg
	} else if (name == "combo") {
		console.log("loading combo " + arg)
		args["c"] = arg
	} else {
		var stash = name + ":" + arg
		args["topxid"] = topxid[stash]
	}
	return args
}
function refreshhonks(btn) {
	btn.innerHTML = "refreshing"
	btn.disabled = true
	var args = hydrargs()
	var stash = curpagestate.name + ":" + curpagestate.arg
	args["topxid"] = topxid[stash]
	get("/hydra?" + encode(args), function(xhr) {
		var lenhonks = fillinhonks(xhr)
		btn.innerHTML = "refresh"
		btn.disabled = false
		btn.parentElement.children[1].innerHTML = " " + lenhonks + " new"
	})
}
function statechanger(evt) {
	var data = evt.state
	if (!data) {
		return
	}
	switchtopage(data.name, data.arg)
}
function switchtopage(name, arg) {
	var honksonpage = document.getElementById("honksonpage")
	var holder = honksonpage.children[0]
	holder.remove()
	// if not convoy, save current page
	if (curpagestate.name != "convoy") {
		var stash = curpagestate.name + ":" + curpagestate.arg
		honksforpage[stash] = holder
	}
	curpagestate.name = name
	curpagestate.arg = arg
	// get the holder for the target page
	var stash = name + ":" + arg
	holder = honksforpage[stash]
	if (holder) {
		honksonpage.prepend(holder)
	} else {
		// or create one and fill it
		honksonpage.prepend(document.createElement("div"))
		var args = hydrargs()
		get("/hydra?" + encode(args), fillinhonks)
	}
	var topmenu = document.getElementById("topmenu")
	topmenu.open = false
}
function newpagestate(name, arg) {
	return { "name": name, "arg": arg }
}
function pageswitcher(name, arg) {
	return function(evt) {
		console.log("switching to", name +":"+arg)
		if (name == curpagestate.name && arg == curpagestate.arg) {
			console.log("skipping nav")
			return false
		}
		switchtopage(name, arg)
		var url = evt.srcElement.href
		history.pushState(newpagestate(name, arg), "some title", url)
		return false
	}
}
function relinklinks() {
	var els = document.getElementsByClassName("convoylink")
	for (var i = 0; i < els.length; i++) {
		els[i].onclick = pageswitcher("convoy", els[i].text)
	}
	els = document.getElementsByClassName("combolink")
	for (var i = 0; i < els.length; i++) {
		els[i].onclick = pageswitcher("combo", els[i].text)
	}
}
(function() {
	var el = document.getElementById("homelink")
	el.onclick = pageswitcher("home", "")
	el = document.getElementById("atmelink")
	el.onclick = pageswitcher("atme", "")
	el = document.getElementById("firstlink")
	el.onclick = pageswitcher("first", "")
	relinklinks()
	window.onpopstate = statechanger
	history.replaceState(curpagestate, "some title", "")
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
