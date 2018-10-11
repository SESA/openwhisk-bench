function main(args) { var spin=0; var count = 0; if(args.spin) spin=args.spin; var max = 1<<spin; for (var line=1; line<max; line++) { count++; } return {done:true, c:count}; }
