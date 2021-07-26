import moment from "../dist/moment.mjs";

/**
 * @alias frontend.websocket
 */
class WebsocketSubscriber {
  /**
   * The websocket
   * @param {} frontend
   */
  constructor(frontend) {
    this.store = frontend.store;
    this.frontend = frontend;
    // Get subscriptions by key
    this.subscriptions = {};

    // Set up the websocket
    let wsproto = "wss:";
    if (location.protocol == "http:") {
      wsproto = "ws:";
    }

    this.loc =
      wsproto + "//" + location.host + location.pathname + "api/events";

    // The websocket server might be disabled for non-logged-in users
    this.retryConnect = frontend.info.user != null;

    this.resetTimeout = 200;
    this.retryTimeout = 200;
    this.retryTimeoutDelta = 1000;

    // Whether the socket is open, and when it was connected. This allows
    // the frontend to check if it needs to query for stuff
    this.isopen = false;

    frontend.worker.addHandler("websocket_subscribe", (ctx, msg) =>
      this.subscribe(msg.key, msg.event, (e) =>
        frontend.worker.postMessage("websocket_event", {
          key: msg.key,
          event: e,
        })
      )
    );
    frontend.worker.addHandler("websocket_unsubscribe", (ctx, msg) =>
      this.unsubscribe(msg.key)
    );

    this.connect();
  }
  connect() {
    console.vlog(`Connecting to websocket ${this.loc}`);
    this.ws = new WebSocket(this.loc);

    this.ws.onopen = () => this.onopen();
    this.ws.onmessage = (m) => this.fire(m);
    this.ws.onclose = (e) => this.onclose(e);
  }
  onopen() {
    console.vlog("Websocket open");
    this.isopen = true;
    this.retryTimeout = this.resetTimeout;

    this.send({
      cmd: "subscribe",
      event: "*",
      user: this.frontend.info.user.username,
    });

    /* In the future, should also handle events by other users

        Object.values(this.subscriptions).forEach((s) => {
            let m = {
                cmd: "subscribe",
                ...s.event
            }
            console.vlog("<-", m);
            this.ws.send(JSON.stringify(m))
        });
        */

    // Set the websocket frontend time
    let m = moment();
    this.store.commit("setWebsocket", m);
    this.frontend.worker.postMessage("websocket_status", m.unix());
  }
  onclose(e) {
    console.vlog("Websocket closed");
    this.isopen = false;
    // Set the websocket as disconnected
    this.store.commit("setWebsocket", null);
    this.frontend.worker.postMessage("websocket_status", null);
    if (this.retryConnect) {
      setTimeout(() => this.connect(), this.retryTimeout);
      this.retryTimeout += this.retryTimeoutDelta;
      return;
    }
    console.vlog("Not retrying to connect.");
  }

  fire(e) {
    e = JSON.parse(e.data);
    console.vlog("->", e);
    Object.values(this.subscriptions)
      .filter((s) => {
        s = s.event;
        // Oh boy, we need to check if the given subscription should be given the event
        if (s.event != e.event && s.event != "*") return false;
        if (
          s.object !== undefined &&
          s.object != "*" &&
          (e.object === undefined || s.object != e.object)
        )
          return false;
        if (
          s.app !== undefined &&
          s.app != "*" &&
          (e.app === undefined || s.app != e.app)
        )
          return false;
        if (
          s.user !== undefined &&
          s.user != "*" &&
          (e.user === undefined || s.user != e.user)
        )
          return false;
        if (
          s.plugin !== undefined &&
          (e.plugin === undefined || e.plugin != s.plugin)
        )
          return false;
        if (s.key !== undefined && (e.key === undefined || e.key != s.key))
          return false;
        return true;
      })
      .forEach((s) => s.callback(e));
  }

  send(m) {
    if (this.isopen) {
      console.vlog("<-", m);
      this.ws.send(JSON.stringify(m));
    }
  }

  subscribe(key, event, callback) {
    this.subscriptions[key] = {
      event: event,
      callback: callback,
    };

    /*
    this.send({
      cmd: "subscribe",
      ...event,
    });
    */
  }

  unsubscribe(key) {
    if (this.subscriptions[key] !== undefined) {
      let k = this.subscriptions[key];
      delete this.subscriptions[key];
      /*
      this.send({
        cmd: "unsubscribe",
        ...k.event,
      });
      */
    }
  }
}

export default WebsocketSubscriber;
