// Plays the deck that build.py put into this page: one slide at a time, with
// the click steps, the slide transitions and the live canvas scenes of the
// published deck. It needs nothing but this file's page; no network.
//
//   right arrow, space, click     next step or slide
//   left arrow, click at the left previous step or slide
//   Home, End                     first and last slide
//   F                             full screen
//   N                             speaker notes
(function () {
  'use strict';

  var DECK = JSON.parse(document.getElementById('deck').textContent);
  var SLIDES = DECK.slides;
  var W = 1920, H = 1080;

  // How long a slide takes to leave, as in the Slides runtime.
  var LEAVE_MS = { fade: 320, push: 420, magic: 480 };
  var LEAVE_EASING = 'cubic-bezier(0.33, 0.01, 0.2, 1)';

  // Where a click step comes in from.
  var STEP_MS = 500;
  var STEP_EASING = 'cubic-bezier(0.2, 0.7, 0.2, 1)';
  var STEP_FROM = {
    fade: 'none', rise: 'translateY(48px)', drop: 'translateY(-48px)',
    left: 'translateX(-72px)', right: 'translateX(72px)', scale: 'scale(0.8)', pop: 'scale(0.5)'
  };
  var POP_EASING = 'cubic-bezier(0.34, 1.56, 0.64, 1)';

  var ARROWS = {
    'arrow-right': '0,30 60,30 60,10 100,50 60,90 60,70 0,70',
    'arrow-left': '100,30 40,30 40,10 0,50 40,90 40,70 100,70',
    'arrow-up': '30,100 30,40 10,40 50,0 90,40 70,40 70,100',
    'arrow-down': '30,0 30,60 10,60 50,100 90,60 70,60 70,0'
  };

  var stage = document.getElementById('stage');
  var notes = document.getElementById('notes');
  var hint = document.getElementById('hint');

  var current = -1;     // index of the slide on the stage
  var slide = null;     // its <section>
  var shown = 0;        // how many of its click steps are visible
  var finishLeaving = null; // ends a transition that is still running

  // ---- building a slide -------------------------------------------------

  function isPinned(el) { return el.style.position === 'absolute'; }

  function isHidden(el) { return el.style.visibility === 'hidden'; }

  // build returns the <section> of slide i, ready to go on the stage.
  function build(i) {
    var holder = document.createElement('template');
    holder.innerHTML = SLIDES[i].html;
    var section = holder.content.firstElementChild;

    // The runtime paints a slide's children in source order, so a backdrop
    // that comes before the text lies under it. A browser paints pinned
    // boxes over flowing text; send those leading backdrops behind it.
    var children = Array.prototype.slice.call(section.children);
    for (var k = 0; k < children.length && isPinned(children[k]); k++) {
      children[k].style.zIndex = '-1';
    }

    section.querySelectorAll('x-icon').forEach(function (icon) {
      icon.innerHTML = DECK.icons[icon.getAttribute('name')] || '';
    });

    section.querySelectorAll('x-shape').forEach(function (shape) {
      var points = ARROWS[shape.getAttribute('kind')];
      if (!points) return;
      var fill = shape.style.backgroundColor || 'currentColor';
      shape.style.background = 'none';
      shape.innerHTML = '<svg viewBox="0 0 100 100" preserveAspectRatio="none"><polygon points="' + points + '" fill="' + fill + '"/></svg>';
    });

    section.querySelectorAll('a').forEach(function (link) {
      link.target = '_blank';
      link.rel = 'noopener';
    });

    section.steps = clickSteps(section);

    return section;
  }

  // clickSteps groups the elements that wait for a click, in click order.
  // data-build-in="rise 2" comes in with the second click.
  function clickSteps(section) {
    var byOrder = {};
    section.querySelectorAll('[data-build-in]').forEach(function (el) {
      var words = el.getAttribute('data-build-in').trim().split(/\s+/);
      var order = parseInt(words[1], 10) || 1;
      (byOrder[order] = byOrder[order] || []).push({ el: el, effect: words[0] });
    });

    return Object.keys(byOrder).map(Number).sort(function (a, b) { return a - b; })
      .map(function (order) { return byOrder[order]; });
  }

  // showSteps makes the first n click steps visible, without animation.
  function showSteps(section, n) {
    section.steps.forEach(function (group, index) {
      group.forEach(function (item) { item.el.style.visibility = index < n ? '' : 'hidden'; });
    });
  }

  function playStep(group) {
    group.forEach(function (item) {
      item.el.style.visibility = '';
      var opacity = getComputedStyle(item.el).opacity;
      item.el.animate(
        [{ opacity: 0, transform: STEP_FROM[item.effect] || 'none' }, { opacity: opacity, transform: 'none' }],
        { duration: STEP_MS, easing: item.effect === 'pop' ? POP_EASING : STEP_EASING }
      );
    });
  }

  // startScenes puts every canvas scene of the slide into a frame of its
  // own, where its script runs apart from this page. A new frame starts the
  // scene from its beginning.
  function startScenes(section) {
    section.querySelectorAll('x-embed').forEach(function (embed) {
      var frame = document.createElement('iframe');
      frame.setAttribute('sandbox', 'allow-scripts');
      frame.setAttribute('scrolling', 'no');
      frame.setAttribute('tabindex', '-1');
      frame.title = 'Animation';
      frame.srcdoc = '<!doctype html><meta charset="utf-8">' + DECK.embeds[Number(embed.getAttribute('data-embed'))];
      embed.appendChild(frame);
    });
  }

  // ---- moving between slides --------------------------------------------

  // show puts slide i on the stage. direction is 1 for forward, -1 for back
  // and 0 for a jump, which has no transition.
  function show(i, direction) {
    if (i < 0 || i >= SLIDES.length || i === current) return;
    if (finishLeaving) finishLeaving();

    var leaving = slide, from = current, back = direction < 0;
    var arriving = build(i);

    // Going back lands on a slide as it was left: with every step shown.
    shown = back ? arriving.steps.length : 0;
    showSteps(arriving, shown);

    stage.insertBefore(arriving, stage.firstChild); // under the slide that leaves
    startScenes(arriving);

    current = i;
    slide = arriving;

    if (leaving) {
      // A slide says how it is left. Going back plays the transition of
      // the slide that is returned to.
      var kind = direction === 0 ? null : SLIDES[back ? i : from].transition;
      if (kind && LEAVE_MS[kind]) leave(leaving, arriving, kind, back);
      else leaving.remove();
    }

    describe();
  }

  function opacityOf(el) { return getComputedStyle(el).opacity; }

  // boxOf is where an element sits on its slide, in slide pixels.
  function boxOf(el, section) {
    var a = el.getBoundingClientRect(), b = section.getBoundingClientRect(), scale = b.width / W;

    return { x: (a.left - b.left) / scale, y: (a.top - b.top) / scale, w: a.width / scale, h: a.height / scale };
  }

  function sameBox(a, b) {
    return Math.abs(a.x - b.x) < 0.5 && Math.abs(a.y - b.y) < 0.5 && Math.abs(a.w - b.w) < 0.5 && Math.abs(a.h - b.h) < 0.5;
  }

  function boxFrame(box, opacity) {
    return {
      left: box.x + 'px', top: box.y + 'px', right: 'auto', bottom: 'auto',
      width: box.w + 'px', height: box.h + 'px', transform: 'none', opacity: opacity
    };
  }

  // leave plays the transition from one slide to the next and then takes
  // the old slide off the stage.
  function leave(leaving, arriving, kind, back) {
    var ms = LEAVE_MS[kind];
    var hold = { duration: ms, easing: LEAVE_EASING, fill: 'forwards' };
    var once = { duration: ms, easing: LEAVE_EASING };
    var playing = [], veil = null;

    leaving.style.pointerEvents = 'none';
    // The runtime draws a leaving slide without its live scenes.
    leaving.querySelectorAll('x-embed').forEach(function (embed) { embed.style.visibility = 'hidden'; });

    if (kind === 'fade') {
      playing.push(leaving.animate([{ opacity: 1 }, { opacity: 0 }], hold));
    } else if (kind === 'push') {
      var shift = back ? -W : W;
      playing.push(leaving.animate([{ transform: 'translateX(0)' }, { transform: 'translateX(' + -shift + 'px)' }], hold));
      playing.push(arriving.animate([{ transform: 'translateX(' + shift + 'px)' }, { transform: 'translateX(0)' }], once));
    } else {
      veil = magicMove(leaving, arriving, hold, once, playing);
    }

    var done = false;
    var timer = null;

    finishLeaving = function () {
      if (done) return;
      done = true;
      clearTimeout(timer);
      playing.forEach(function (animation) { animation.cancel(); });
      leaving.remove();
      if (veil) veil.remove();
      finishLeaving = null;
    };

    timer = setTimeout(finishLeaving, ms + 30);
  }

  // magicMove is the "magic" transition. What both slides have, by id,
  // moves from its old place to its new one. Everything else on the old
  // slide fades out and everything else on the new one fades in.
  function magicMove(leaving, arriving, hold, once, playing) {
    var veil = null;
    var oldBackground = getComputedStyle(leaving).backgroundColor;

    if (oldBackground !== getComputedStyle(arriving).backgroundColor) {
      veil = document.createElement('div');
      veil.style.cssText = 'position:absolute;left:0;top:0;width:100%;height:100%;z-index:-2;background:' + oldBackground;
      arriving.insertBefore(veil, arriving.firstChild);
      playing.push(veil.animate([{ opacity: 1 }, { opacity: 0 }], hold));
    }

    leaving.style.background = 'transparent';

    var counterparts = {};
    Array.prototype.forEach.call(arriving.children, function (el) {
      if (el.id && !isHidden(el)) counterparts[el.id] = el;
    });

    var moved = [];

    Array.prototype.forEach.call(leaving.children, function (old) {
      if (isHidden(old)) return;

      var next = old.id ? counterparts[old.id] : null;
      if (!next) {
        playing.push(old.animate([{ opacity: opacityOf(old) }, { opacity: 0 }], hold));

        return;
      }

      moved.push(next);

      var from = boxOf(old, leaving), to = boxOf(next, arriving);
      var fromOpacity = opacityOf(old), toOpacity = opacityOf(next);

      old.style.visibility = 'hidden';

      if (!sameBox(from, to)) {
        playing.push(next.animate([boxFrame(from, fromOpacity), boxFrame(to, toOpacity)], once));
      } else if (fromOpacity !== toOpacity) {
        playing.push(next.animate([{ opacity: fromOpacity }, { opacity: toOpacity }], once));
      }
    });

    Array.prototype.forEach.call(arriving.children, function (el) {
      if (el === veil || isHidden(el) || moved.indexOf(el) !== -1) return;
      playing.push(el.animate([{ opacity: 0 }, { opacity: opacityOf(el) }], once));
    });

    return veil;
  }

  function next() {
    if (shown < slide.steps.length) {
      playStep(slide.steps[shown]);
      shown++;
    } else {
      show(current + 1, 1);
    }
  }

  function previous() {
    if (shown > 0) {
      shown--;
      showSteps(slide, shown);
    } else {
      show(current - 1, -1);
    }
  }

  // ---- the page around the slides ---------------------------------------

  // describe keeps the address, the title and the notes in step with the slide.
  function describe() {
    var place = (current + 1) + ' / ' + SLIDES.length;
    history.replaceState(null, '', '#' + (current + 1));
    document.title = DECK.title + ' · ' + place;

    notes.textContent = '';
    var heading = document.createElement('b');
    heading.textContent = 'Slide ' + place;
    notes.appendChild(heading);
    notes.appendChild(document.createTextNode(SLIDES[current].notes || 'No notes for this slide.'));
  }

  function fit() {
    var scale = Math.min(innerWidth / W, innerHeight / H);
    stage.style.transform =
      'translate(' + (innerWidth - W * scale) / 2 + 'px,' + (innerHeight - H * scale) / 2 + 'px) scale(' + scale + ')';
  }

  function fromAddress() {
    var number = parseInt(location.hash.slice(1), 10);

    return number >= 1 && number <= SLIDES.length ? number - 1 : 0;
  }

  function toggleFullScreen() {
    if (document.fullscreenElement) document.exitFullscreen();
    else if (document.documentElement.requestFullscreen) document.documentElement.requestFullscreen();
  }

  addEventListener('keydown', function (event) {
    if (event.ctrlKey || event.metaKey || event.altKey) return;

    switch (event.key) {
      case 'ArrowRight': case 'ArrowDown': case 'PageDown': case ' ': case 'Enter': next(); break;
      case 'ArrowLeft': case 'ArrowUp': case 'PageUp': case 'Backspace': previous(); break;
      case 'Home': show(0, 0); break;
      case 'End': show(SLIDES.length - 1, 0); break;
      case 'f': case 'F': toggleFullScreen(); break;
      case 'n': case 'N': notes.hidden = !notes.hidden; break;
      default: return;
    }

    event.preventDefault();
  });

  addEventListener('click', function (event) {
    if (event.target.closest('a') || event.target.closest('#notes')) return;
    if (event.clientX < innerWidth * 0.2) previous();
    else next();
  });

  addEventListener('hashchange', function () { show(fromAddress(), 0); });
  addEventListener('resize', fit);

  // The pointer gets out of the way when it rests.
  var resting = null;
  addEventListener('mousemove', function () {
    document.body.classList.remove('idle');
    clearTimeout(resting);
    resting = setTimeout(function () { document.body.classList.add('idle'); }, 2500);
  });

  fit();
  show(fromAddress(), 0);
  setTimeout(function () { hint.classList.add('gone'); }, 5000);
})();
