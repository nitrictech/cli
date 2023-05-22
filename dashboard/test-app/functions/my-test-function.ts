import { api, bucket, collection, faas, schedule, topic } from "@nitric/sdk";

const firstApi = api("first-api");
const secondApi = api("second-api");

const myBucket = bucket("test-bucket").for("reading", "writing", "deleting");

interface Doc {
  firstCount: number;
  secondCount: number;
}

const col = collection<Doc>("test-collection").for("writing", "reading");

firstApi.get("/schedule-count", async (ctx) => {
  try {
    const data = await col.doc("schedule-count").get();

    return ctx.res.json(data);
  } catch (e) {
    return ctx.res.json({
      firstCount: 0,
      secondCount: 0,
    } as Doc);
  }
});

firstApi.get("/topic-count", async (ctx) => {
  try {
    const data = await col.doc("topic-count").get();

    return ctx.res.json(data);
  } catch (e) {
    return ctx.res.json({
      firstCount: 0,
      secondCount: 0,
    } as Doc);
  }
});

// test all methods
firstApi.post("/all-methods", async (ctx) => ctx);
firstApi.put("/all-methods", async (ctx) => ctx);
firstApi.patch("/all-methods", async (ctx) => ctx);
firstApi.get("/all-methods", async (ctx) => ctx);
firstApi.delete("/all-methods", async (ctx) => ctx);
firstApi.options("/all-methods", async (ctx) => ctx);

firstApi.get("/path-test/:name", async (ctx) => {
  const { name } = ctx.req.params;

  ctx.res.body = `Hello ${name}`;

  return ctx;
});

firstApi.get("/header-test", async (ctx) => {
  const data = ctx.req.headers;

  return ctx.res.json({
    headers: data,
  });
});

firstApi.get("/query-test", async (ctx) => {
  const data = ctx.req.query;

  return ctx.res.json({
    queryParams: data,
  });
});

firstApi.post("/json-test", async (ctx) => {
  const data = ctx.req.json();

  return ctx.res.json({
    requestData: data,
  });
});

secondApi.get("/content-type-html", (ctx) => {
  const html = `
    <html>
      <head>
        <title>My Web Page</title>
      </head>
      <body>
        <h1>Welcome to my web page</h1>
        <p>This is some sample HTML content.</p>
      </body>
    </html>
  `;
  ctx.res.headers = { "content-type": ["text/html"] };
  ctx.res.body = html.trim();

  return ctx;
});

secondApi.get("/content-type-css", (ctx) => {
  const css = `
    body {
      font-family: Arial, sans-serif;
      background-color: #f1f1f1;
    }

    h1 {
      color: blue;
    }

    p {
      color: green;
    }
  `;
  ctx.res.headers = { "content-type": ["text/css"] };
  ctx.res.body = css.trim();

  return ctx;
});

secondApi.get("/content-type-image", (ctx) => {
  const svgData = `
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
      <rect width="100%" height="100%" fill="#ff0000"/>
      <text x="50%" y="50%" font-size="48" fill="#ffffff" text-anchor="middle">SVG</text>
    </svg>
  `;
  ctx.res.headers = { "content-type": ["image/svg+xml"] };
  ctx.res.body = svgData;

  return ctx;
});

secondApi.get("/content-type-xml", (ctx) => {
  const xmlData = `
    <?xml version="1.0" encoding="UTF-8"?>
    <data>
      <user>
        <name>John Doe</name>
        <email>john.doe@example.com</email>
      </user>
      <user>
        <name>Jane Smith</name>
        <email>jane.smith@example.com</email>
      </user>
    </data>
  `;
  ctx.res.headers = { "content-type": ["text/xml"] };
  ctx.res.body = xmlData.trim();

  return ctx;
});

secondApi.get("/content-type-binary", (ctx) => {
  const xmlData = `
    <?xml version="1.0" encoding="UTF-8"?>
    <data>
      <user>
        <name>John Doe</name>
        <email>john.doe@example.com</email>
      </user>
      <user>
        <name>Jane Smith</name>
        <email>jane.smith@example.com</email>
      </user>
    </data>
  `;
  ctx.res.headers = { "content-type": ["application/xml"] };
  ctx.res.body = xmlData.trim();

  return ctx;
});

secondApi.get("/image-from-bucket", async (ctx) => {
  const image = await myBucket.file("images/photo.jpg").read();

  ctx.res.body = image;
  ctx.res.headers = { "Content-Type": ["image/jpeg"] };
  return ctx;
});

secondApi.put("/image-from-bucket", async (ctx) => {
  const imageData = ctx.req.data;

  await myBucket.file("images/photo.jpg").write(imageData);

  return ctx;
});

secondApi.put("/very-nested-files", async (ctx) => {
  const { fileName } = ctx.req.query;
  const data = ctx.req.data;

  await myBucket.file(`5/4/3/2/1/${fileName}`).write(data);

  return ctx;
});

secondApi.delete("/image-from-bucket", async (ctx) => {
  await myBucket.file("images/photo.jpg").delete();

  return ctx;
});

schedule("process-tests").every("5 minutes", async (ctx) => {
  try {
    const data = await col.doc("schedule-count").get();

    await col.doc("schedule-count").set({
      ...data,
      firstCount: data.firstCount + 1,
    });
  } catch (e) {
    await col.doc("schedule-count").set({
      firstCount: 1,
      secondCount: 0,
    });
  }
});

schedule("process-tests-2").every("5 minutes", async (ctx) => {
  try {
    const data = await col.doc("schedule-count").get();

    await col.doc("schedule-count").set({
      ...data,
      secondCount: data.secondCount + 1,
    });
  } catch (e) {
    await col.doc("schedule-count").set({
      firstCount: 0,
      secondCount: 1,
    });
  }
});

topic("subscribe-tests").subscribe(async (ctx) => {
  try {
    const data = await col.doc("topic-count").get();

    await col.doc("topic-count").set({
      ...data,
      firstCount: data.firstCount + 1,
    });
  } catch (e) {
    await col.doc("topic-count").set({
      firstCount: 1,
      secondCount: 0,
    });
  }
});

topic("subscribe-tests-2").subscribe(async (ctx) => {
  try {
    const data = await col.doc("topic-count").get();

    await col.doc("topic-count").set({
      ...data,
      secondCount: data.secondCount + 1,
    });
  } catch (e) {
    await col.doc("topic-count").set({
      firstCount: 0,
      secondCount: 1,
    });
  }
});
