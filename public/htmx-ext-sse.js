// https://unpkg.com/htmx-ext-sse@2.2.3/dist/sse.min.js
(function(){var g;htmx.defineExtension("sse",{init:function(e){g=e;if(htmx.createEventSource==undefined){htmx.createEventSource=t}},getSelectors:function(){return["[sse-connect]","[data-sse-connect]","[sse-swap]","[data-sse-swap]"]},onEvent:function(e,t){var r=t.target||t.detail.elt;switch(e){case"htmx:beforeCleanupElement":var n=g.getInternalData(r);var s=n.sseEventSource;if(s){g.triggerEvent(r,"htmx:sseClose",{source:s,type:"nodeReplaced"});n.sseEventSource.close()}return;case"htmx:afterProcessNode":i(r)}}});function t(e){return new EventSource(e,{withCredentials:true})}function a(n){if(g.getAttributeValue(n,"sse-swap")){var s=g.getClosestMatch(n,v);if(s==null){return null}var e=g.getInternalData(s);var a=e.sseEventSource;var t=g.getAttributeValue(n,"sse-swap");var r=t.split(",");for(var i=0;i<r.length;i++){const u=r[i].trim();const c=function(e){if(l(s)){return}if(!g.bodyContains(n)){a.removeEventListener(u,c);return}if(!g.triggerEvent(n,"htmx:sseBeforeMessage",e)){return}f(n,e.data);g.triggerEvent(n,"htmx:sseMessage",e)};g.getInternalData(n).sseEventListener=c;a.addEventListener(u,c)}}if(g.getAttributeValue(n,"hx-trigger")){var s=g.getClosestMatch(n,v);if(s==null){return null}var e=g.getInternalData(s);var a=e.sseEventSource;var o=g.getTriggerSpecs(n);o.forEach(function(t){if(t.trigger.slice(0,4)!=="sse:"){return}var r=function(e){if(l(s)){return}if(!g.bodyContains(n)){a.removeEventListener(t.trigger.slice(4),r)}htmx.trigger(n,t.trigger,e);htmx.trigger(n,"htmx:sseMessage",e)};g.getInternalData(n).sseEventListener=r;a.addEventListener(t.trigger.slice(4),r)})}}function i(e,t){if(e==null){return null}if(g.getAttributeValue(e,"sse-connect")){var r=g.getAttributeValue(e,"sse-connect");if(r==null){return}n(e,r,t)}a(e)}function n(r,e,n){var s=htmx.createEventSource(e);s.onerror=function(e){g.triggerErrorEvent(r,"htmx:sseError",{error:e,source:s});if(l(r)){return}if(s.readyState===EventSource.CLOSED){n=n||0;n=Math.max(Math.min(n*2,128),1);var t=n*500;window.setTimeout(function(){i(r,n)},t)}};s.onopen=function(e){g.triggerEvent(r,"htmx:sseOpen",{source:s});if(n&&n>0){const t=r.querySelectorAll("[sse-swap], [data-sse-swap], [hx-trigger], [data-hx-trigger]");for(let e=0;e<t.length;e++){a(t[e])}n=0}};g.getInternalData(r).sseEventSource=s;var t=g.getAttributeValue(r,"sse-close");if(t){s.addEventListener(t,function(){g.triggerEvent(r,"htmx:sseClose",{source:s,type:"message"});s.close()})}}function l(e){if(!g.bodyContains(e)){var t=g.getInternalData(e).sseEventSource;if(t!=undefined){g.triggerEvent(e,"htmx:sseClose",{source:t,type:"nodeMissing"});t.close();return true}}return false}function f(t,r){g.withExtensions(t,function(e){r=e.transformResponse(r,null,t)});var e=g.getSwapSpecification(t);var n=g.getTarget(t);g.swap(n,r,e)}function v(e){return g.getInternalData(e).sseEventSource!=null}})();
