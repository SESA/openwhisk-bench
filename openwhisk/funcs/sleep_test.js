function main(args) {
    var sleep_time;
    if (args.time) sleep_time = args.time;
    else sleep_time = 1;
    var start = new Date().getTime();
    var wait_util = start + (sleep_time * 1000); 
    while (new Date().getTime() < wait_util){}
}
