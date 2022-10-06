var puppeteer = require('puppeteer');

const track = async () => {
    const trackingCode = process.argv[2]

    const browser = await puppeteer.launch({
        headless: true
    });

    const page = await browser.newPage();

    await page.setUserAgent('Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36');
    await page.goto('https://parcelsapp.com/widget');
    await page.setRequestInterception(true)

    page.on('request', interceptedRequest => {
        if (interceptedRequest.url().endsWith("parcelsapp.com/api/v2/parcels")) {
            const rawBody = interceptedRequest.postData()
            const body = JSON.parse(rawBody)
            body.se = "2575x1407,2560x1392,2560x882,no,Win32,Gecko,Mozilla,Netscape,n/a,true,no,Google Inc. (AMD),ANGLE (AMD, Radeon HD 3200 Graphics Direct3D11 vs_5_0 ps_5_0),152,13,3156752409"
            interceptedRequest.continue({postData: JSON.stringify(body)});
            return
        }

        interceptedRequest.continue();
    });

    await page.type('#track-input', trackingCode);
    await page.click('#track-button');

    const response = await page.waitForResponse("https://parcelsapp.com/api/v2/parcels")
    var result = await response.text()

    await browser.close();

    return result
}
track().then((res) => console.log(res))
