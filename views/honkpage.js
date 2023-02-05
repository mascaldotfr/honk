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
	el.innerHTML = "💥"
	post("/bonk", encode({"js": "2", "CSRF": csrftoken, "xid": xid}))
	el.className = "unbonk"
	el.onclick = function() { unbonk(el, xid) }
}
function unbonk(el, xid) {
	el.innerHTML = "🚀"
	post("/zonkit", encode({"CSRF": csrftoken, "wherefore": "unbonk", "what": xid}))
	el.className = "bonk"
	el.onclick = function() { bonk(el, xid) }
}
function zonkit(el, xid) {
	el.innerHTML = "☠"
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
		window.alert("You have new notifications!")
	} else {
		mecount.innerHTML = ""
	}

	var frontload = true
	if (curpagestate.name == "convoy") {
		frontload = false
	}

	var holder = document.getElementById("honksonpage")
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
function relinklinks() {
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
		} else if (el.classList.contains("zonk")) {
			el.onclick = function() {
				zonkit(el, xid);
			}
		}
	})
}
function showhonkform(elem, rid, hname) {
	var form = lehonkform
	form.style = "display: block"
	if (elem) {
		form.remove()
		elem.parentElement.insertAdjacentElement('afterend', form)
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


const debounce = (func, wait) => {
  let timeout;

  return function executedFunction(...args) {
    const later = () => {
      clearTimeout(timeout);
      func(...args);
    };

    clearTimeout(timeout);
    timeout = setTimeout(later, wait);
  };
};

function cleanmentionssuggestions() {
	datalist = document.getElementById("allthepeople")
	if (!datalist) {
		return
	}
	// should use replaceChildren but keep old code for compat
	while (datalist.firstChild) {
		datalist.firstChild.remove()
	}
	return datalist
}

var refreshmentions = debounce ( function () {
	search = document.getElementById("mentionpeople")
	if (!search) {
		return
	}
	datalist = cleanmentionssuggestions();

	if (search.value.length < 3 || !datalist) {
		return
	}

	get("/searchxonkers?q=" + encodeURIComponent(search.value), function(xhr) {
			if (xhr.status == 200) {
				xhr.response.sort().forEach(x => {
					var option = document.createElement('option');
					option.value = x
					datalist.appendChild(option)
				} )
			} },
			function(xhr, e) { } )
}, 300)

function addmention() {
	honknoise = document.getElementById("honknoise")
	mention = document.getElementById("mentionpeople")
	search = document.getElementById("mentionpeople")
	if (!honknoise || !mention || !search) {
		return
	}
	[start, end] = [honknoise.selectionStart, honknoise.selectionEnd]
	honknoise.setRangeText("@" + mention.value + " ", start, end, "end")

	cleanmentionssuggestions()
	search.value = ""
}

// init
(function() {
	var me = document.currentScript;
	csrftoken = me.dataset.csrf
	curpagestate.name = me.dataset.pagename
	curpagestate.arg = me.dataset.pagearg
	tophid[curpagestate.name + ":" + curpagestate.arg] = me.dataset.tophid
	servermsgs[curpagestate.name + ":" + curpagestate.arg] = me.dataset.srvmsg

	var refreshbtn = document.getElementById("refreshhonks")
	if (refreshbtn) {
		refreshbtn.onclick = function() {
			refreshhonks(refreshbtn)
		}
	}

	var mentionpeople = document.getElementById("mentionpeople")
	if (mentionpeople) {
		mentionpeople.addEventListener("keyup", refreshmentions)
	}
	var mentionbtn = document.getElementById("mentionthem")
	if (mentionbtn) {
		mentionbtn.onclick = function(e) {
			e.preventDefault()
			addmention()
		}
	}

	document.getElementById("honkingtime").onclick = function() {
		return showhonkform()
	}
	document.querySelector("button[name=cancel]").onclick = cancelhonking

	relinklinks()
	window.onpopstate = statechanger
	history.replaceState(curpagestate, "some title", "")

	if ('serviceWorker' in navigator) { navigator.serviceWorker.register('/sw.js') }
})();
