var puppeteer = require('puppeteer');

const track = async () => {
    const trackingCode = process.argv[2]

    const browser = await puppeteer.launch({
        headless: false
    });

    const page = await browser.newPage();

    await page.setUserAgent('Chrome/106.0.0.0');
    await page.goto('https://parcelsapp.com/widget');
    await page.type('#track-input', trackingCode);
    await page.click('#track-button');

    const response = await page.waitForResponse("https://parcelsapp.com/api/v2/parcels")
    var result = await response.text()

    await browser.close();

    return result
}
track().then((res) => console.log(res))
