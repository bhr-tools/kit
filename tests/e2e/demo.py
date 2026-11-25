"""Browser tests of the kit primitives on the demo page (cmd/demo)."""
import os
import time

from playwright.sync_api import sync_playwright, expect

BASE = os.environ["BASE_URL"]
AUTH = os.environ["AUTH_URL"]
OUT = os.environ.get("OUT", "tests/e2e/out")
failures = []


def check(name, fn):
    try:
        fn()
        print(f"ok   {name}")
    except Exception as e:  # noqa: BLE001 — report every failure, then exit 1
        failures.append(name)
        print(f"FAIL {name}: {e}")


def rows(page):
    return page.locator("#list .row").all_inner_texts()


def fresh(browser, **kw):
    ctx = browser.new_context(**kw)
    page = ctx.new_page()
    page.goto(BASE)
    expect(page.locator("#list .row")).to_have_count(4)
    return ctx, page


def keyboard(browser):
    ctx, page = fresh(browser)
    page.keyboard.press("j")
    page.keyboard.press("j")
    expect(page.locator(".row.cursor")).to_have_text("Charlie")
    page.keyboard.press("g")
    page.keyboard.press("g")
    expect(page.locator(".row.cursor")).to_have_text("Alpha")
    page.keyboard.press("G")
    expect(page.locator(".row.cursor")).to_have_text("Delta")
    page.keyboard.press("K")
    assert rows(page) == ["Alpha", "Bravo", "Delta", "Charlie"], rows(page)
    # ? help is generated from the table.
    page.keyboard.press("?")
    expect(page.locator(".modal.help")).to_be_visible()
    expect(page.locator(".help-row", has_text="Move to folder").locator("kbd")).to_have_text("m")
    page.wait_for_timeout(400)
    page.screenshot(path=f"{OUT}/help.png")
    page.keyboard.press("Escape")
    expect(page.locator(".modal")).to_have_count(0)
    # AZERTY digit: é is 2.
    page.evaluate("window.dispatchEvent(new KeyboardEvent('keydown', { key: 'é', bubbles: true }))")
    expect(page.locator("#title")).to_have_text("Archive")
    ctx.close()


def undo(browser):
    ctx, page = fresh(browser)
    page.keyboard.press("j")
    page.keyboard.press("d")
    expect(page.locator(".toast")).to_contain_text("Item deleted")
    assert rows(page) == ["Alpha", "Charlie", "Delta"]
    page.keyboard.press("Space")
    page.keyboard.press("j")
    page.keyboard.press("Space")
    page.keyboard.press("d")
    assert rows(page) == ["Alpha"], rows(page)
    page.keyboard.press("u")
    assert rows(page) == ["Alpha", "Charlie", "Delta"], rows(page)
    page.keyboard.press("u")
    assert rows(page) == ["Alpha", "Bravo", "Charlie", "Delta"], rows(page)
    page.keyboard.press("u")
    expect(page.locator(".toast")).to_contain_text("Nothing to undo")
    ctx.close()


def dialogs(browser):
    ctx, page = fresh(browser)
    page.keyboard.press("e")
    field = page.locator(".modal input")
    expect(field).to_be_focused()
    field.fill("Alpine")
    page.keyboard.press("Enter")
    expect(page.locator(".row").first).to_have_text("Alpine")
    # Typed right after the shortcut, keys reach the field.
    page.keyboard.press("a")
    page.keyboard.type("Echo", delay=0)
    page.keyboard.press("Enter")
    expect(page.locator(".row").last).to_have_text("Echo")
    # Confirm: Esc cancels.
    page.keyboard.press("x")
    expect(page.locator(".modal .button.danger")).to_be_focused()
    page.keyboard.press("Escape")
    expect(page.locator("#list .row")).to_have_count(5)
    # Menu and context menu.
    page.keyboard.press("m")
    page.locator(".menu-item", has_text="Archive").click()
    expect(page.locator("#list .row")).to_have_count(4)
    page.locator(".row", has_text="Bravo").click(button="right")
    page.locator(".menu-item", has_text="Delete").click()
    expect(page.locator("#list .row")).to_have_count(3)
    ctx.close()


def drag(browser):
    ctx, page = fresh(browser)
    first = page.locator(".row", has_text="Alpha").bounding_box()
    third = page.locator(".row", has_text="Charlie").bounding_box()
    page.mouse.move(first["x"] + 20, first["y"] + first["height"] / 2)
    page.mouse.down()
    page.mouse.move(first["x"] + 20, first["y"] + 20, steps=3)
    page.mouse.move(third["x"] + 20, third["y"] + third["height"] + 2, steps=8)
    expect(page.locator(".drop-line")).to_be_visible()
    page.wait_for_timeout(400)
    page.screenshot(path=f"{OUT}/drop-line.png")
    page.mouse.up()
    expect(page.locator(".drop-line")).to_be_hidden()
    assert rows(page) == ["Bravo", "Charlie", "Alpha", "Delta"], rows(page)
    # Onto a folder: it lights up, the row moves there.
    row = page.locator(".row", has_text="Delta").bounding_box()
    folder = page.locator('.folder[data-folder="Archive"]').bounding_box()
    page.mouse.move(row["x"] + 20, row["y"] + row["height"] / 2)
    page.mouse.down()
    page.mouse.move(folder["x"] + 20, folder["y"] + folder["height"] / 2, steps=10)
    expect(page.locator(".folder.drop-target")).to_have_count(1)
    page.mouse.up()
    assert rows(page) == ["Bravo", "Charlie", "Alpha"], rows(page)
    page.keyboard.press("u")
    assert "Delta" in rows(page)
    ctx.close()


def phone(browser):
    for scheme in ("dark", "light"):
        ctx, page = fresh(browser, viewport={"width": 390, "height": 844}, is_mobile=True, has_touch=True, color_scheme=scheme)
        expect(page.locator(".tabbar")).to_be_visible()
        expect(page.locator('.tab[data-tab="Inbox"]')).to_have_class("tab active")
        page.keyboard.press("Enter")
        modal = page.locator(".modal")
        expect(modal).to_be_visible()
        time.sleep(0.4)
        box = modal.bounding_box()
        assert box["x"] == 0 and box["width"] == 390 and box["height"] > 800, box
        expect(page.locator(".modal-close .close-text")).to_have_text("Back")
        expect(page.locator(".tabbar")).to_be_hidden()
        page.wait_for_timeout(400)
        page.screenshot(path=f"{OUT}/phone-page-{scheme}.png")
        page.locator(".modal-close").click()
        expect(modal).to_have_count(0)
        page.keyboard.press("m")
        expect(page.locator(".popover.sheet .sheet-title")).to_have_text("Move to")
        page.wait_for_timeout(400)
        page.screenshot(path=f"{OUT}/phone-sheet-{scheme}.png")
        ctx.close()


def themes(browser):
    """Rosé Pine main / Dawn backgrounds; accent foam in dark, pine in light."""
    want = {"dark": ("rgb(25, 23, 36)", "rgb(156, 207, 216)"), "light": ("rgb(250, 244, 237)", "rgb(40, 105, 131)")}
    for scheme, (bg, accent) in want.items():
        ctx, page = fresh(browser, color_scheme=scheme)
        got_bg = page.evaluate("getComputedStyle(document.body).backgroundColor")
        assert got_bg == bg, (scheme, got_bg)
        page.keyboard.press("e")
        focus_border = page.evaluate("getComputedStyle(document.querySelector('.modal input')).borderColor")
        assert focus_border == accent, (scheme, focus_border)
        page.wait_for_timeout(400)
        page.screenshot(path=f"{OUT}/desktop-{scheme}.png")
        page.keyboard.press("Escape")
        ctx.close()


def lock(browser):
    ctx = browser.new_context()
    page = ctx.new_page()
    page.goto(AUTH)
    expect(page.locator(".modal-title")).to_have_text("Welcome to Kit demo")
    fields = page.locator(".modal input[type=password]")
    fields.nth(0).fill("abc")
    fields.nth(1).fill("abc")
    page.locator(".modal button[type=submit]").click()
    expect(page.locator(".form-error")).to_contain_text("at least 4")
    fields.nth(0).fill("secret")
    fields.nth(1).fill("secret")
    page.locator(".modal button[type=submit]").click()
    expect(page.locator(".modal")).to_have_count(0)
    page.reload()
    page.locator(".lock-button").click()
    page.locator(".menu-item", has_text="Lock").click()
    expect(page.locator(".modal-title")).to_have_text("Kit demo is locked")
    page.locator(".modal input[type=password]").fill("wrong")
    page.keyboard.press("Enter")
    expect(page.locator(".form-error")).to_contain_text("Incorrect password")
    page.locator(".modal input[type=password]").fill("secret")
    page.keyboard.press("Enter")
    expect(page.locator(".modal")).to_have_count(0)
    ctx.close()


def offline(browser):
    ctx = browser.new_context()
    page = ctx.new_page()
    page.goto(BASE)
    page.evaluate("navigator.serviceWorker.register('/sw.js')")
    page.wait_for_function("navigator.serviceWorker.controller !== null || navigator.serviceWorker.ready.then(() => true)")
    time.sleep(1)
    ctx.set_offline(True)
    page.reload()
    expect(page.locator("#list .row")).to_have_count(4)
    ctx.close()


with sync_playwright() as p:
    browser = p.chromium.launch()
    for name, fn in [("keyboard", keyboard), ("undo", undo), ("dialogs", dialogs), ("drag", drag),
                     ("phone", phone), ("themes", themes), ("lock", lock), ("offline", offline)]:
        check(name, lambda fn=fn: fn(browser))
    browser.close()

if failures:
    raise SystemExit(f"{len(failures)} failed: {', '.join(failures)}")
print("all passed")
