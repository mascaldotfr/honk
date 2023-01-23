var csrftoken = ""
var honksforpage = { }
var curpagestate = { name: "", arg : "" }
var tophid = { }
var servermsgs = { }

function encode(hash) {
        var s = []
        for (var key in hash) {
                var val = hash[key]
                s.push(encodeURIComponent(key) + "=" + encodeURIComponent(val))
        }
        return s.join("&")
}
function post(url, data) {
	var x = new XMLHttpRequest()
	x.open("POST", url)
	x.timeout = 30 * 1000
	x.setRequestHeader("Content-Type", "application/x-www-form-urlencoded")
	x.send(data)
}
function get(url, whendone, whentimedout) {
	var x = new XMLHttpRequest()
	x.open("GET", url)
	x.timeout = 15 * 1000
	x.responseType = "json"
	x.onload = function() { whendone(x) }
	if (whentimedout) {
		x.ontimeout = function(e) { whentimedout(x, e) }
	}
	x.send()
}
function bonk(el, xid) {
	el.innerHTML = "bonked"
	el.disabled = true
	post("/bonk", encode({"js": "2", "CSRF": csrftoken, "xid": xid}))
	return false
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
function flogit(el, how, xid) {
	var s = how
	if (s[s.length-1] != "e") { s += "e" }
	s += "d"
	if (s == "untaged") s = "untagged"
	if (s == "reacted") s = "badonked"
	el.innerHTML = s
	el.disabled = true
	post("/zonkit", encode({"CSRF": csrftoken, "wherefore": how, "what": xid}))
}

var lehonkform = document.getElementById("honkform")
var lehonkbutton = document.getElementById("honkingtime")

function oldestnewest(btn) {
	var els = document.getElementsByClassName("glow")
	if (els.length) {
		els[els.length-1].scrollIntoView()
	}
}
function removeglow() {
	var els = document.getElementsByClassName("glow")
	while (els.length) {
		els[0].classList.remove("glow")
	}
}

function fillinhonks(xhr, glowit) {
	var resp = xhr.response
	var stash = curpagestate.name + ":" + curpagestate.arg
	tophid[stash] = resp.Tophid
	var doc = document.createElement( 'div' );
	doc.innerHTML = resp.Srvmsg
	var srvmsg = doc
	doc = document.createElement( 'div' );
	doc.innerHTML = resp.Honks
	var honks = doc.children

	var mecount = document.getElementById("mecount")
	if (resp.MeCount) {
		mecount.innerHTML = "(" + resp.MeCount + ")"
	} else {
		mecount.innerHTML = ""
	}

	var srvel = document.getElementById("srvmsg")
	while (srvel.children[0]) {
		srvel.children[0].remove()
	}
	srvel.prepend(srvmsg)

	var frontload = true
	if (curpagestate.name == "convoy") {
		frontload = false
	}

	var honksonpage = document.getElementById("honksonpage")
	var holder = honksonpage.children[0]
	var lenhonks = honks.length
	for (var i = honks.length; i > 0; i--) {
		var h = honks[i-1]
		if (glowit)
			h.classList.add("glow")
		if (frontload) {
			holder.prepend(h)
		} else {
			holder.append(h)
		}
	}
	var newhonks = document.getElementsByClassName("glow")
	if (newhonks.length > 0) {
		let oldesthonk = newhonks[newhonks.length - 1]
		oldesthonk.scrollIntoView({ behavior: "auto", block: "end" })
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
		args["c"] = arg
	} else if (name == "honker") {
		args["xid"] = arg
	} else if (name == "user") {
		args["uname"] = arg
	}
	return args
}
function refreshhonks(btn) {
	removeglow()
	btn.innerHTML = "refreshing"
	btn.disabled = true
	var args = hydrargs()
	var stash = curpagestate.name + ":" + curpagestate.arg
	args["tophid"] = tophid[stash]
	get("/hydra?" + encode(args), function(xhr) {
		btn.innerHTML = "refresh"
		btn.disabled = false
		if (xhr.status == 200) {
			fillinhonks(xhr, true)
		}
	}, function(xhr, e) {
		btn.innerHTML = "refresh"
		btn.disabled = false
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
	var stash = curpagestate.name + ":" + curpagestate.arg
	var honksonpage = document.getElementById("honksonpage")
	var holder = honksonpage.children[0]
	holder.remove()
	var srvel = document.getElementById("srvmsg")
	var msg = srvel.children[0]
	if (msg) {
		msg.remove()
		servermsgs[stash] = msg
	}

	honksforpage[stash] = holder

	curpagestate.name = name
	curpagestate.arg = arg
	// get the holder for the target page
	var stash = name + ":" + arg
	holder = honksforpage[stash]
	if (holder) {
		honksonpage.prepend(holder)
		msg = servermsgs[stash]
		if (msg) {
			srvel.prepend(msg)
		}
	} else {
		// or create one and fill it
		honksonpage.prepend(document.createElement("div"))
		var args = hydrargs()
		get("/hydra?" + encode(args), function(xhr) {
			if (xhr.status == 200) {
				fillinhonks(xhr, false)
			}
		}, function(xhr, e) {
		})
	}
}
function newpagestate(name, arg) {
	return { "name": name, "arg": arg }
}
function pageswitcher(name, arg) {
	return function(evt) {
		var topmenu = document.getElementById("topmenu")
		topmenu.open = false
		if (name == curpagestate.name && arg == curpagestate.arg) {
			return false
		}
		switchtopage(name, arg)
		var url = evt.srcElement.href
		if (!url) {
			url = evt.srcElement.parentElement.href
		}
		history.pushState(newpagestate(name, arg), "some title", url)
		window.scrollTo(0, 0)
		return false
	}
}
function relinklinks() {
	var els = document.getElementsByClassName("convoylink")
	while (els.length) {
		els[0].onclick = pageswitcher("convoy", 
			decodeURIComponent(els[0].search.replace(/\?c=/, "")));
		els[0].classList.remove("convoylink")
	}
	els = document.getElementsByClassName("combolink")
	while (els.length) {
		els[0].onclick = pageswitcher("combo", 
			decodeURIComponent(els[0].search.replace(/\?c=/, "")));
		els[0].classList.remove("combolink")
	}
	els = document.getElementsByClassName("honkerlink")
	while (els.length) {
		var el = els[0]
		var xid = el.getAttribute("data-xid")
		el.onclick = pageswitcher("honker", xid)
		el.classList.remove("honkerlink")
	}
	els = document.querySelectorAll("#honksonpage article button")
	els.forEach(function(el) {
		var honk = el.closest("article")
		var convoy = honk.dataset.convoy
		var hname = honk.dataset.hname
		var xid = honk.dataset.xid
		var id = Number(honk.dataset.id)

		if (!(id > 0)) {
			console.error("could not determine honk id")
			return
		}

		if (el.classList.contains("unbonk")) {
			el.onclick = function() {
				unbonk(el, xid);
			}
		} else if (el.classList.contains("bonk")) {
			el.onclick = function() {
				bonk(el, xid)
			}
		} else if (el.classList.contains("honkback")) {
			el.onclick = function() {
				return showhonkform(el, xid, hname)
			}
		} else if (el.classList.contains("mute")) {
			el.onclick = function() {
				muteit(el, convoy);
			}
		} else if (el.classList.contains("evenmore")) {
			var more = document.querySelector("#evenmore"+id);
			el.onclick = function() {
				more.classList.toggle("hide");
			}
		} else if (el.classList.contains("zonk")) {
			el.onclick = function() {
				zonkit(el, xid);
			}
		} else if (el.classList.contains("flogit-deack")) {
			el.onclick = function() {
				flogit(el, "deack", xid);
			}
		} else if (el.classList.contains("flogit-ack")) {
			el.onclick = function() {
				flogit(el, "ack", xid);
			}
		} else if (el.classList.contains("flogit-unsave")) {
			el.onclick = function() {
				flogit(el, "unsave", xid);
			}
		} else if (el.classList.contains("flogit-save")) {
			el.onclick = function() {
				flogit(el, "save", xid);
			}
		} else if (el.classList.contains("flogit-untag")) {
			el.onclick = function() {
				flogit(el, "untag", xid);
			}
		} else if (el.classList.contains("flogit-react")) {
			el.onclick = function() {
				flogit(el, "react", xid);
			}
		} else if (el.classList.contains("playit")) {
			var noise = el.dataset.noise
			var wonk = el.dataset.wonk
			el.onclick = function() {
				playit(el, noise, wonk, xid)
			}
		}
	})
}
(function() {
	var el = document.getElementById("homelink")
	el.onclick = pageswitcher("home", "")
	el = document.getElementById("atmelink")
	el.onclick = pageswitcher("atme", "")
	relinklinks()
	window.onpopstate = statechanger
	history.replaceState(curpagestate, "some title", "")
})();
(function() {
	hideelement("donkdescriptor")
})();
function showhonkform(elem, rid, hname) {
	var form = lehonkform
	form.style = "display: block"
	if (elem) {
		form.remove()
		elem.parentElement.parentElement.parentElement.insertAdjacentElement('beforebegin', form)
	} else {
		hideelement(lehonkbutton)
		elem = document.getElementById("honkformhost")
		elem.insertAdjacentElement('afterend', form)
	}
	var ridinput = document.getElementById("ridinput")
	if (rid) {
		ridinput.value = rid
		if (hname) {
			honknoise.value = hname + " "
		} else {
			honknoise.value = ""
		}
	} else {
		ridinput.value = ""
		honknoise.value = ""
	}
	var updateinput = document.getElementById("updatexidinput")
	updateinput.value = ""
	document.getElementById("honknoise").focus()
	return false
}
function cancelhonking() {
	hideelement(lehonkform)
	showelement(lehonkbutton)
}
function showelement(el) {
	if (typeof(el) == "string")
		el = document.getElementById(el)
	if (!el) return
	el.style.display = "flex"
}
function hideelement(el) {
	if (typeof(el) == "string")
		el = document.getElementById(el)
	if (!el) return
	el.style.display = "none"
}
function updatedonker() {
	var el = document.getElementById("donker")
	el.children[1].textContent = el.children[0].value.slice(-20)
	var el = document.getElementById("donkdescriptor")
	el.style.display = ""
	var el = document.getElementById("saveddonkxid")
	el.value = ""
}

// init
(function() {
	var me = document.currentScript;
	csrftoken = me.dataset.csrf
	curpagestate.name = me.dataset.pagename
	curpagestate.arg = me.dataset.pagearg
	tophid[curpagestate.name + ":" + curpagestate.arg] = me.dataset.tophid
	servermsgs[curpagestate.name + ":" + curpagestate.arg] = me.dataset.srvmsg

	var totop = document.querySelector(".nophone")
	if (totop) {
		totop.onclick = function() {
			window.scrollTo(0,0)
		}
	}

	var refreshbox = document.getElementById("refreshbox")
	if (refreshbox) {
		refreshbox.querySelectorAll("button").forEach(function(el) {
			if (el.classList.contains("refresh")) {
				el.onclick = function() {
					refreshhonks(el)
				}
			} else if (el.classList.contains("scrolldown")) {
				el.onclick = function() {
					oldestnewest(el)
				}
			}
		})

		if (me.dataset.srvmsg == "one honk maybe more") {
			hideelement(refreshbox)
		}
	}

	var td = document.getElementById("timedescriptor")
	document.getElementById("honkingtime").onclick = function() {
		return showhonkform()
	}
	document.querySelector("button[name=cancel]").onclick = cancelhonking
})();
